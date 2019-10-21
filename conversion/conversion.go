// Copyright 2019 Tomas Machalek <tomas.machalek@gmail.com>
// Copyright 2019 Institute of the Czech National Corpus,
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

package conversion

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	// AppTypeKontext defines a universal storage identifier for KonText
	AppTypeKontext = "kontext"

	// AppTypeSyd defines a universal storage identifier for SyD
	AppTypeSyd = "syd"

	// AppTypeMorfio defines a universal storage identifier for Morfio
	AppTypeMorfio = "morfio"

	// AppTypeKwords defines a universal storage identifier for Kwords
	AppTypeKwords = "kwords"

	// AppTypeTreq defines a universal storage identifier for Treq
	AppTypeTreq = "treq"

	// AppTypeSke defines a universal storage identifier for Treq
	AppTypeSke = "ske"

	// AppTypeWag defines a universal storage identifier for Word at a Glance
	AppTypeWag = "wag"

	// AppTypeCalc defines a universal storage identifier for Calc
	AppTypeCalc = "calc"

	// AppTypeLists defines a universal storage identifier for Lists
	AppTypeLists = "lists"
)

// LineParsingError informs that we failed to parse a line we
// actually wanted to parse. This means no need to stop whole
// parsing but still it may be important to inform user about
// that.
type LineParsingError struct {
	LineNumber int
	Message    string
}

func (m LineParsingError) Error() string {
	return fmt.Sprintf("line %d: %s", m.LineNumber, m.Message)
}

// NewLineParsingError is a constructor for LineParsingError
func NewLineParsingError(lineNumber int, message string) LineParsingError {
	return LineParsingError{LineNumber: lineNumber, Message: message}
}

// InputRecord describes a common behavior for objects extracted
// from an application log of any UCNK app.
type InputRecord interface {
	GetTime() time.Time
	GetClientIP() net.IP
	AgentIsLoggable() bool
}

// GeoDataRecord represents a full client geographical
// position information as provided by GeoIP database
type GeoDataRecord struct {
	ContinentCode string     `json:"continent_code"`
	CountryCode2  string     `json:"country_code2"`
	CountryCode3  string     `json:"country_code3"`
	CountryName   string     `json:"country_name"`
	IP            string     `json:"ip"`
	Latitude      float32    `json:"latitude"`
	Longitude     float32    `json:"longitude"`
	Location      [2]float32 `json:"location"`
	Timezone      string     `json:"timezone"`
}

// OutputRecord describes a common behavior for records ready to
// be stored to the storage with a defined type. Implementation
// details are up to concrete implementations but these functions are
// required by the 'processing template'.
type OutputRecord interface {
	SetLocation(countryName string, latitude float32, longitude float32, timezone string)

	// ToJSON creates an object suitable for storing to ElasticSearch, CouchDB and other
	// document-oriented databases
	ToJSON() ([]byte, error)

	// ToInfluxDB creates two maps: 1) tags, 2) values as defined
	// by InfluxDB architecture. These can be directly saved via
	// a respective InfluxDB client.
	ToInfluxDB() (tags map[string]string, values map[string]interface{})

	// Create an idempotent unique identifier of the record.
	// This can be typically acomplished by hashing the original
	// log record.
	GetID() string

	// Return app type as defined by an external convention
	// (e.g. for UCNK: kontext, syd, morfio, treq,...)
	GetType() string

	// Get time of the log record
	GetTime() time.Time
}

// LogItemTransformer defines a general object able to transform
// an input log record to an output one.
type LogItemTransformer interface {
	Transform(logRec InputRecord, recType string, anonymousUsers []int) (OutputRecord, error)
}

// UserBelongsToList tests whether a provided user can be
// found in a provided array of users.
func UserBelongsToList(userID int, anonymousUsers []int) bool {
	for _, v := range anonymousUsers {
		if v == userID {
			return true
		}
	}
	return false
}

// TimezoneToInt returns number of minutes to add/subtract to apply
// to UTC to get actual local time reprezented by 'tz'.
func TimezoneToInt(tz string) (int, error) {
	sgn := 1
	if tz[0] == '-' {
		sgn = -1

	} else if tz[0] != '+' {
		return 0, fmt.Errorf("Cannot parse %s as timezone value", tz)
	}
	items := strings.Split(tz[1:], ":")
	if len(items) != 2 {
		return 0, fmt.Errorf("Cannot parse %s as timezone value", tz)
	}
	v1, err := strconv.Atoi(items[0])
	if err != nil {
		return 0, err
	}
	v2, err := strconv.Atoi(items[1])
	if err != nil {
		return 0, err
	}
	return sgn * (60*v1 + v2), nil
}

// ImportBool imports typical bool formats (as supported by Go) with
// additional support for an empty space, 'yes' and 'no' strings.
func ImportBool(v, keyName string) (bool, error) {
	if v == "" {
		return false, nil
	}
	if v == "yes" {
		return true, nil
	}
	if v == "no" {
		return false, nil
	}
	ans, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("invalid data for %s: %s", keyName, v)
	}
	return ans, nil
}

// ConvertDatetimeString imports ISO 8601 datetime string. In case
// of a parsing error, "zero" time instance is created.
func ConvertDatetimeString(datetime string) time.Time {
	if t, err := time.Parse("2006-01-02T15:04:05-07:00", datetime); err == nil {
		return t
	}
	return time.Time{}
}

func ConvertAccessLogDatetimeString(datetime string) time.Time {
	if t, err := time.Parse("02/Jan/2006:15:04:05 -0700", datetime); err == nil {
		return t
	}
	return time.Time{}
}
