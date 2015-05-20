// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bigquery

import (
	"fmt"

	bq "google.golang.org/api/bigquery/v2"
)

type loadDestination interface {
	customizeLoadDst(conf *bq.JobConfigurationLoad)
}

type loadSource interface {
	customizeLoadSrc(conf *bq.JobConfigurationLoad)
}

type loadOption interface {
	customizeLoad(conf *bq.JobConfigurationLoad)
}

// A DestinationSchema must be supplied when loading data from Google Cloud Storage into a non-existent table.
// Caveat: DestinationSchema is not required if the data being loaded is a datastore backup.
func DestinationSchema(schema Schema) Option { return destSchema(schema) }

type destSchema Schema

func (opt destSchema) implementsOption() {}

func (opt destSchema) customizeLoad(conf *bq.JobConfigurationLoad) {
	var fields []*bq.TableFieldSchema
	for _, f := range opt {
		fields = append(fields, f.proto())
	}
	if len(fields) > 0 {
		conf.Schema = &bq.TableSchema{Fields: fields}
	}
}

// MaxBadRecords returns an Option that sets the maximum number of bad records that will be ignored.
// If this maximum is exceeded, the operation will be unsuccessful.
func MaxBadRecords(n int64) Option { return maxBadRecords(n) }

type maxBadRecords int64

func (opt maxBadRecords) implementsOption() {}

func (opt maxBadRecords) customizeLoad(conf *bq.JobConfigurationLoad) {
	conf.MaxBadRecords = int64(opt)
}

// AllowJaggedRows returns an Option that causes missing trailing optional columns to be tolerated in CSV data.  Missing values are treated as nulls.
func AllowJaggedRows() Option { return allowJaggedRows{} }

type allowJaggedRows struct{}

func (opt allowJaggedRows) implementsOption() {}

func (opt allowJaggedRows) customizeLoad(conf *bq.JobConfigurationLoad) {
	conf.AllowJaggedRows = true
}

// AllowQuotedNewlines returns an Option that allows quoted data sections containing newlines in CSV data.
func AllowQuotedNewlines() Option { return allowQuotedNewlines{} }

type allowQuotedNewlines struct{}

func (opt allowQuotedNewlines) implementsOption() {}

func (opt allowQuotedNewlines) customizeLoad(conf *bq.JobConfigurationLoad) {
	conf.AllowQuotedNewlines = true
}

// IgnoreUnknownValues returns an Option that causes values not matching the schema to be tolerated.
// Unknown values are ignored. For CSV this ignores extra values at the end of a line.
// For JSON this ignores named values that do not match any column name.
// If this Option is not used, records containing unknown values are treated as bad records.
// The MaxBadRecords Option can be used to customize how bad records are handled.
func IgnoreUnknownValues() Option { return ignoreUnknownValues{} }

type ignoreUnknownValues struct{}

func (opt ignoreUnknownValues) implementsOption() {}

func (opt ignoreUnknownValues) customizeLoad(conf *bq.JobConfigurationLoad) {
	conf.IgnoreUnknownValues = true
}

func load(c jobInserter, dst Destination, src Source, options ...Option) (*Job, error) {
	payload := &bq.JobConfigurationLoad{}

	d := dst.(loadDestination)
	s := src.(loadSource)

	d.customizeLoadDst(payload)
	s.customizeLoadSrc(payload)

	for _, opt := range options {
		o, ok := opt.(loadOption)
		if !ok {
			return nil, fmt.Errorf("Option not applicable to dst/src pair: %#v", opt)
		}
		o.customizeLoad(payload)
	}

	return c.insertJob(&bq.Job{
		Configuration: &bq.JobConfiguration{
			Load: payload,
		},
	})
}
