/*
 * ZLint Copyright 2018 Regents of the University of Michigan
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not
 * use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
 * implied. See the License for the specific language governing
 * permissions and limitations under the License.
 */

/********************************************************************
4.1.2.5.2.  GeneralizedTime
The generalized time type, GeneralizedTime, is a standard ASN.1 type
for variable precision representation of time.  Optionally, the
GeneralizedTime field can include a representation of the time
differential between local and Greenwich Mean Time.

For the purposes of this profile, GeneralizedTime values MUST be
expressed in Greenwich Mean Time (Zulu) and MUST include seconds
(i.e., times are YYYYMMDDHHMMSSZ), even where the number of seconds
is zero.  GeneralizedTime values MUST NOT include fractional seconds.
********************************************************************/

package lints

import (
	"github.com/zmap/zcrypto/x509"
	"github.com/zmap/zlint/util"
)

type generalizedNotZulu struct {
	date1Gen bool
	date2Gen bool
}

func (l *generalizedNotZulu) Initialize() error {
	return nil
}

func (l *generalizedNotZulu) CheckApplies(c *x509.Certificate) bool {
	firstDate, secondDate := util.GetTimes(c)
	beforeTag, afterTag := util.FindTimeType(firstDate, secondDate)
	l.date1Gen = beforeTag == 24
	l.date2Gen = afterTag == 24
	return l.date1Gen || l.date2Gen
}

func (l *generalizedNotZulu) Execute(c *x509.Certificate) *LintResult {
	date1, date2 := util.GetTimes(c)
	if l.date1Gen {
		if date1.Bytes[len(date1.Bytes)-1] != 'Z' {
			return &LintResult{Status: Error}
		}
	}
	if l.date2Gen {
		if date2.Bytes[len(date2.Bytes)-1] != 'Z' {
			return &LintResult{Status: Error}
		}
	}
	return &LintResult{Status: Pass}
}

func init() {
	RegisterLint(&Lint{
		Name:          "e_generalized_time_not_in_zulu",
		Description:   "Generalized time values MUST be expressed in Greenwich Mean Time (Zulu)",
		Citation:      "RFC 5280: 4.1.2.5.2",
		Source:        RFC5280,
		EffectiveDate: util.RFC2459Date,
		Lint:          &generalizedNotZulu{},
	})
}