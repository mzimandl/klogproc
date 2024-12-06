// Copyright 2024 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2024 Institute of the Czech National Corpus,
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

package treqapi

import (
	"fmt"
	"klogproc/scripting"
	"klogproc/servicelog"
	"klogproc/servicelog/treq"
	"strconv"

	lua "github.com/yuin/gopher-lua"
)

// Transformer converts a Treq log record to a destination format
type Transformer struct {
	ExcludeIPList  servicelog.ExcludeIPList
	AnonymousUsers []int
}

func (t *Transformer) AppType() string {
	return servicelog.AppTypeTreq
}

func (t *Transformer) Transform(
	logRecord servicelog.InputRecord,
	tzShiftMin int,
) (servicelog.OutputRecord, error) {
	tLogRecord, ok := logRecord.(*InputRecord)
	if !ok {
		panic(servicelog.ErrFailedTypeAssertion)
	}

	userID := -1
	if tLogRecord.UserID != "-" {
		if uid, err := strconv.Atoi(tLogRecord.UserID); err == nil {
			userID = uid

		} else {
			return nil, fmt.Errorf(
				"failed to convert user ID '%s': %w", tLogRecord.UserID, err)
		}
	}

	out := &treq.OutputRecord{
		Type:        "treq",
		IsAPI:       true,
		QLang:       tLogRecord.From,
		SecondLang:  tLogRecord.To,
		IPAddress:   tLogRecord.IP,
		UserID:      tLogRecord.UserID,
		IsAnonymous: userID == -1 || servicelog.UserBelongsToList(userID, t.AnonymousUsers),
		IsRegexp:    tLogRecord.Regex,
		IsCaseInsen: tLogRecord.CI,
		IsMultiWord: tLogRecord.Multiword,
		IsLemma:     tLogRecord.Lemma,
	}
	out.SetTime(tLogRecord.GetTime(), tzShiftMin)
	out.ID = out.GenerateDeterministicID()
	return out, nil
}

func (t *Transformer) SetOutputProperty(rec servicelog.OutputRecord, name string, value lua.LValue) error {
	return scripting.ErrScriptingNotSupported
}

func (t *Transformer) HistoryLookupItems() int {
	return 0
}

func (t *Transformer) Preprocess(
	rec servicelog.InputRecord, prevRecs servicelog.ServiceLogBuffer,
) []servicelog.InputRecord {
	if t.ExcludeIPList.Excludes(rec) {
		return []servicelog.InputRecord{}
	}
	return []servicelog.InputRecord{rec}
}