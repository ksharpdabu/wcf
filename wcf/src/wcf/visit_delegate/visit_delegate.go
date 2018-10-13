package visit_delegate

import "wcf/visit"
import _ "wcf/visit/visit_sqlite"
import _ "wcf/visit/visit_json"

func Get(name string) (visit.Visitor, error) {
	return visit.Get(name)
}
