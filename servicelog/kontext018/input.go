// Copyright 2023 Institute of the Czech National Corpus,
//                Faculty of Arts, Charles University
// Copyright 2023 Martin Zimandl <martin.zimandl@gmail.com>
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

package kontext018

import (
	"fmt"
	"klogproc/servicelog"
	"net"
	"time"
)

func getSliceOfStrings(data interface{}, key string) ([]string, bool) {
	v, ok := data.(map[string]interface{})
	if !ok {
		return []string{}, false
	}
	v2, ok := v[key]
	if !ok {
		return []string{}, false
	}
	v3, ok := v2.([]interface{})
	if !ok {
		return []string{}, false
	}
	ans := make([]string, len(v3))
	for i, s := range v3 {
		ans[i] = string(fmt.Sprintf("%v", s))
	}
	return ans, true
}

type ExceptionInfo struct {
	ID    string   `json:"id"`
	Type  string   `json:"type"`
	Stack []string `json:"stack"`
}

// Request is a simple representation of
// HTTP request metadata used in KonText logging
type Request struct {
	HTTPForwardedFor string `json:"HTTP_X_FORWARDED_FOR"`
	HTTPUserAgent    string `json:"HTTP_USER_AGENT"`
	HTTPRemoteAddr   string `json:"HTTP_REMOTE_ADDR"`
	RemoteAddr       string `json:"REMOTE_ADDR"`
	// HTTPIsWebApp specifies whether the request (even API one)
	// is from CNC's web application and thus not considered
	// true "API use"
	HTTPIsWebApp string `json:"HTTP_X_IS_WEB_APP"`
}

// InputRecord represents Kontext query log
type InputRecord struct {
	Logger         string                 `json:"logger"`
	Level          string                 `json:"level"`
	Date           string                 `json:"date"`
	Message        string                 `json:"message"`
	Exception      ExceptionInfo          `json:"exception"`
	UserID         int                    `json:"user_id"`
	ProcTime       float64                `json:"proc_time"`
	Action         string                 `json:"action"`
	IsIndirectCall bool                   `json:"is_indirect_call"`
	IsAPI          bool                   `json:"is_api"`
	Request        Request                `json:"request"`
	Args           map[string]interface{} `json:"args"`
	Error          servicelog.ErrorRecord `json:"error"`
	isProcessable  bool
}

// GetTime returns record's time as a Golang's Time
// instance. Please note that the value is truncated
// to seconds.
func (rec *InputRecord) GetTime() time.Time {
	if rec.isProcessable {
		if rec.Date[len(rec.Date)-1] == 'Z' {
			return servicelog.ConvertDatetimeStringWithMillisNoTZ(rec.Date[:len(rec.Date)-1] + "000")
		}
		return servicelog.ConvertDatetimeStringWithMillis(rec.Date)
	}
	return time.Time{}
}

// GetClientIP returns a client IP no matter in which
// part of the record it was found
// (e.g. REMOTE_ADDR vs. HTTP_REMOTE_ADDR vs. HTTP_FORWARDED_FOR)
func (rec *InputRecord) GetClientIP() net.IP {
	if rec.Request.HTTPForwardedFor != "" {
		return net.ParseIP(rec.Request.HTTPForwardedFor)

	} else if rec.Request.HTTPRemoteAddr != "" {
		return net.ParseIP(rec.Request.HTTPRemoteAddr)

	} else if rec.Request.RemoteAddr != "" {
		return net.ParseIP(rec.Request.RemoteAddr)
	}
	return nil
}

func (rec *InputRecord) ShouldBeAnalyzed() bool {
	return rec.Action == "query_submit" || rec.Action == "create_view" ||
		rec.Action == "create_lazy_view" || rec.Action == "wordlist/submit"
	// TODO the list of actions is incomplete
}

func (rec *InputRecord) ClusteringClientID() string {
	return servicelog.GenerateRandomClusteringID()
}

func (rec *InputRecord) ClusterSize() int {
	return 0
}

func (rec *InputRecord) SetCluster(size int) {
}

// GetUserAgent returns a raw HTTP user agent info as provided by the client
func (rec *InputRecord) GetUserAgent() string {
	return rec.Request.HTTPUserAgent
}

// IsProcessable returns true if there was no error in reading the record
func (rec *InputRecord) IsProcessable() bool {
	return rec.isProcessable
}

// GetStringArg fetches a string parameter from
// a special "args" sub-object. The function supports
// nested keys - e.g. {"foo": {"bar": "test"}} can be
// accessed via GetStringArg("foo", "bar")
func (rec *InputRecord) GetStringArg(names ...string) string {
	var val interface{}
	val = rec.Args
	for _, name := range names {
		valmap, ok := val.(map[string]interface{})
		if !ok {
			return ""
		}
		val = valmap[name]
	}
	switch v := val.(type) {
	case string:
		return v
	}
	return ""
}

// HasArg tests whether there is a top-level key matching
// a provided name
func (rec *InputRecord) HasArg(name string) bool {
	_, ok := rec.Args[name]
	return ok
}

// GetIntArg fetches an integer parameter from
// a special "params" sub-object
func (rec *InputRecord) GetIntArg(name string) int {
	switch v := rec.Args[name].(type) {
	case int:
		return v
	}
	return -1
}

func (rec *InputRecord) AllArgs() map[string]any {
	return rec.Args
}

// GetAlignedCorpora returns a list of aligned corpora
// (i.e. not the first corpus but possible other corpora aligned
// with the main one)
func (rec *InputRecord) GetAlignedCorpora() []string {
	corpora, _ := getSliceOfStrings(rec.Args, "corpora")
	if len(corpora) > 0 {
		return corpora[1:]
	}
	return []string{}
}

func (rec *InputRecord) IsSuspicious() bool {
	return false
}
