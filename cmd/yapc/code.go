package main

import (
	"strings"
	"strconv"
)

package main

import (
"fmt"
"strconv"
"strings"
"github.com/imdario/mergo"
)

func main() {

	s := map[string]interface{}{
		"/users/0/name":         "bernard",
		"/users/0/surname":      "balt",
		"/users/0/interests/0":  "diy",
		"/users/0/address/home": "33 creellin",
		"/users/0/interests/1":  "gym",
		"/users/1/name":         "keith",
		"/users/1/surname":      "stubby",
		"/users/1/interests/0":  "nrl",
		"/users/1/interests/1":  "v8",
		"/users/1/address/home": "44 creellin",
	}
	dst := map[string]interface{}{}
	for k, v := range s {
		ks := strings.Split(strings.TrimLeft(k, "/"), "/")
		ks_r := make([]string, len(ks))
		for ksi, ksv := range ks {
			ks_r[len(ks)-(ksi+1)] = ksv
		}

		for _, kv := range ks_r {
			ik, err := strconv.Atoi(kv)
			if err == nil {
				tmp := make([]interface{}, (ik + 1))
				tmp[ik] = v
				v = tmp
				continue
			}

			v = map[string]interface{}{kv: v}

		}

		if err := mergo.Merge(&dst, v); err != nil {

		}
	}

}

