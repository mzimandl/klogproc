// Copyright 2020 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2020 Institute of the Czech National Corpus,
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

package mapka3

import (
	"crypto/sha1"
	"encoding/hex"
	"strconv"
	"time"

	"klogproc/conversion"
	"klogproc/logbuffer"
)

// createID creates an idempotent ID of rec based on its properties.
func createID(rec *OutputRecord) string {
	str := rec.Type + rec.Path + rec.Datetime + rec.IPAddress + rec.UserID
	sum := sha1.Sum([]byte(str))
	return hex.EncodeToString(sum[:])
}

// Transformer converts a source log object into a destination one
type Transformer struct {
	historyLookupSecs int
}

// Transform creates a new OutputRecord out of an existing InputRecord
func (t *Transformer) Transform(
	logRecord *InputRecord,
	recType string,
	tzShiftMin int,
	anonymousUsers []int,
) (*OutputRecord, error) {

	userID := -1

	r := &OutputRecord{
		Type:        recType,
		time:        logRecord.GetTime(),
		Datetime:    logRecord.GetTime().Add(time.Minute * time.Duration(tzShiftMin)).Format(time.RFC3339),
		IPAddress:   logRecord.Extra.IP,
		UserAgent:   logRecord.GetUserAgent(),
		IsAnonymous: userID == -1 || conversion.UserBelongsToList(userID, anonymousUsers),
		IsQuery:     false,
		UserID:      strconv.Itoa(userID),
	}
	r.ID = createID(r)
	r.IsQuery = true
	r.Action = "interaction"
	return r, nil
}

func (t *Transformer) HistoryLookupSecs() int {
	return t.historyLookupSecs
}

func (t *Transformer) Preprocess(
	rec conversion.InputRecord, prevRecs *logbuffer.Storage[conversion.InputRecord],
) conversion.InputRecord {
	return rec
}

// NewTransformer is a default constructor for the Transformer.
// It also loads user ID map from a configured file (if exists).
func NewTransformer(historyLookupSecs int) *Transformer {
	return &Transformer{historyLookupSecs: historyLookupSecs}
}
