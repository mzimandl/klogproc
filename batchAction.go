// Copyright 2017 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2017 Institute of the Czech National Corpus,
//                Faculty of Arts, Charles University
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"klogproc/config"
	"klogproc/conversion"
	"klogproc/load/batch"
	"klogproc/save"
	"klogproc/save/elastic"
	"klogproc/save/influx"
	"klogproc/users"
	"log"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

func runBatchAction(
	conf *config.Main,
	options *ProcessOptions,
	geoDB *geoip2.Reader,
	userMap *users.UserMap,
	analyzer ClientAnalyzer,
	finishEvent chan<- bool,
) {

	lt, err := GetLogTransformer(conf.LogFiles.AppType, conf.LogFiles.Version, userMap)
	if err != nil {
		log.Fatal(err)
	}
	processor := &CNKLogProcessor{
		geoIPDb:        geoDB,
		chunkSize:      conf.ElasticSearch.PushChunkSize,
		appType:        conf.LogFiles.AppType,
		appVersion:     conf.LogFiles.Version,
		logTransformer: lt,
		anonymousUsers: conf.AnonymousUsers,
		clientAnalyzer: analyzer,
	}
	channelWriteES := make(chan *conversion.BoundOutputRecord, conf.ElasticSearch.PushChunkSize*2)
	channelWriteInflux := make(chan *conversion.BoundOutputRecord, conf.InfluxDB.PushChunkSize)
	worklog := batch.NewWorklog(conf.LogFiles.WorklogPath)
	log.Printf("INFO: using worklog %s", conf.LogFiles.WorklogPath)
	if options.worklogReset {
		log.Printf("truncated worklog %v", worklog)
		err := worklog.Reset()
		if err != nil {
			log.Fatalf("FATAL: unable to initialize worklog: %s", err)
		}
	}
	defer worklog.Save()

	var wg sync.WaitGroup
	if options.dryRun || options.analysisOnly {
		ch1 := save.RunWriteConsumer(channelWriteES, !options.analysisOnly)
		go func() {
			for range ch1 {
			}
			wg.Add(1)
		}()
		ch2 := save.RunWriteConsumer(channelWriteInflux, !options.analysisOnly)
		go func() {
			for range ch2 {
			}
			wg.Add(1)
		}()
		log.Print("WARNING: using dry-run mode, output goes to stdout")

	} else {
		ch1 := elastic.RunWriteConsumer(conf.LogFiles.AppType, &conf.ElasticSearch, channelWriteES)
		ch2 := influx.RunWriteConsumer(&conf.InfluxDB, channelWriteInflux)
		go func() {
			for confirm := range ch1 {
				if confirm.Error != nil {
					log.Print("ERROR: failed to save data to ElasticSearch db: ", confirm.Error)
					// TODO
				}
			}
			wg.Add(1)
		}()
		go func() {
			for confirm := range ch2 {
				if confirm.Error != nil {
					log.Print("ERROR: failed to save data to InfluxDB db: ", confirm.Error)
					// TODO
				}
			}
			wg.Add(1)
		}()
	}
	proc := batch.CreateLogFileProcFunc(processor, options.datetimeRange, channelWriteES, channelWriteInflux)
	proc(&conf.LogFiles, worklog.GetLastRecord())
	close(channelWriteES)
	close(channelWriteInflux)
	wg.Wait()
	finishEvent <- true
	log.Printf("INFO: Ignored %d non-loggable entries (bots, static files etc.)", processor.numNonLoggable)

	if options.analysisOnly {
		fmt.Println("Detected bot/script activities:")
		for _, sr := range processor.clientAnalyzer.GetBotCandidates() {
			js, err := sr.ToJSON()
			if err != nil {
				log.Println("ERROR: ", err)
			}
			fmt.Println(string(js))
		}
	}
}