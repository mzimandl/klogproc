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

package mapka2

import (
	"encoding/json"
	"klogproc/servicelog"
	"time"
)

// OutputRecord represents a polished version of Mapka's access log stripped
// of unnecessary attributes
type OutputRecord struct {
	ID          string `json:"-"`
	Type        string `json:"type"`
	Action      string `json:"action"`
	Path        string `json:"path"`
	Datetime    string `json:"datetime"`
	time        time.Time
	IPAddress   string                   `json:"ipAddress"`
	UserAgent   string                   `json:"userAgent"`
	UserID      string                   `json:"userId"`
	IsAnonymous bool                     `json:"isAnonymous"`
	IsQuery     bool                     `json:"isQuery"`
	GeoIP       servicelog.GeoDataRecord `json:"geoip,omitempty"`
	ProcTime    float32                  `json:"procTime"`
}

// SetLocation sets all the location related properties
func (r *OutputRecord) SetLocation(countryName string, latitude float32, longitude float32, timezone string) {
	r.GeoIP.IP = r.IPAddress
	r.GeoIP.CountryName = countryName
	r.GeoIP.Latitude = latitude
	r.GeoIP.Longitude = longitude
	r.GeoIP.Location[0] = r.GeoIP.Longitude
	r.GeoIP.Location[1] = r.GeoIP.Latitude
	r.GeoIP.Timezone = timezone
}

// GetID returns an idempotent ID of the record.
func (r *OutputRecord) GetID() string {
	return r.ID
}

// GetType returns application type identifier
func (r *OutputRecord) GetType() string {
	return r.Type
}

// GetTime returns a creation time of the record
func (r *OutputRecord) GetTime() time.Time {
	return r.time
}

// ToJSON converts data to a JSON document (typically for ElasticSearch)
func (r *OutputRecord) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}
