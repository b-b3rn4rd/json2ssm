package main

import (
	"strings"
	"strconv"
	"fmt"
)

package main

import (
"fmt"
"strconv"
"strings"
)

func main() {
	package main

	import (
		"fmt"
	"strconv"
	"strings"
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

		for k, v := range s {
			//tr := map[string]interface{}{}

			ks := strings.Split(strings.TrimLeft(k, "/"), "/")
			ks_r := make([]string, len(ks))
			for ksi, ksv := range ks {
				ks_r[len(ks)-(ksi+1)] = ksv
			}

			_, err := strconv.Atoi(ks_r[0])
			if err == nil {
				v = []interface{}{v}
			} else {
				v = map[string]interface{}{ks_r[0]:v}
			}

			for _, kv := range ks_r[1:] {
				_, err := strconv.Atoi(kv)
				if err == nil {
					v = []interface{}{v}
					continue
				}

				v = map[string]interface{}{kv:v}


			}
			fmt.Println(v)
		}

	}

}

