[![Go Report Card](https://goreportcard.com/badge/github.com/b-b3rn4rd/json2ssm)](https://goreportcard.com/report/github.com/b-b3rn4rd/json2ssm)  [![Build Status](https://travis-ci.org/b-b3rn4rd/gocfn.svg?branch=master)](https://travis-ci.org/b-b3rn4rd/json2ssm) JSON import/export functions with AWS SSM Parameter Store
================================================
*json2ssm* - provides JSON import, export and delete functionality when working with AWS SSM parameter store while keeping original data types.

Motivation
--------------
AWS SSM Parameter Store is a great service for centrally storing and managing application parameters and secrets.
However, seeding parameters often becomes an adhoc process when parameters are added manually
or provisioned from different scripts which makes it difficult to promote applications between environments.

Using `json2ssm` parameters can be imported from a single or multiple JSON source files, additionally, it also provides export function to recursively retrieve parameters in JSON format for scenarios
when it's easier to work with JSON structure, for example with Jenkins pipelines or inside ansible playbooks.

Examples
----------------

Let's start with an example and import following file into SSM parameter store:

```bash
$ cat ../../pkg/storage/testdata/colors.json
{
  "colors": [
    {
      "color": "black",
      "category": "hue",
      "type": "primary",
      "code": {
        "rgba": [255,255,255,1],
        "hex": "#000"
      }
    },
    {
      "color": "white",
      "category": "value",
      "code": {
        "rgba": [0,0,0,1],
        "hex": "#FFF"
      }
    },
    {
      "color": "red",
      "category": "hue",
      "type": "primary",
      "code": {
        "rgba": [255,0,0,1],
        "hex": "#FF0"
      }
    },
    {
      "color": "blue",
      "category": "hue",
      "type": "primary",
      "code": {
        "rgba": [0,0,255,1],
        "hex": "#00F"
      }
    },
    {
      "color": "yellow",
      "category": "hue",
      "type": "primary",
      "code": {
        "rgba": [255,255,0,1],
        "hex": "#FF0"
      }
    },
    {
      "color": "green",
      "category": "hue",
      "type": "secondary",
      "code": {
        "rgba": [0,255,0,1],
        "hex": "#0F0"
      }
    }
  ]
}
```

```bash
$ json2ssm put-json --json-file ../../pkg/storage/testdata/colors.json
 47 / 47 [=============================================================================>]  100%
 Import has successfully finished, 47 parameters have been (over)written to SSM parameter store. 
```

Retrieve the first color:

```bash
$ json2ssm get-json --path "/colors/0"
 8 / 8 [==============================================================================] 8s
{
 "category": "hue",
 "code": {
  "hex": "#000",
  "rgba": [
   255,
   255,
   255,
   1
  ]
 },
 "color": "black",
 "type": "primary"
}
```

Retrieve the first color's `rgba` value and store it in a file:
```bash
$ json2ssm get-json --path "/colors/0/code/rgba" > rgba.son
 4 / 4 [==============================================================================] 2s
$ cat rgba.json 
  [
   255,
   255,
   255,
   1
  ]
```

Installation
=============
```bash
brew tap b-b3rn4rd/homebrew-tap
brew install json2ssm
```

*Using go get*

```bash
go get github.com/b-b3rn4rd/json2ssm
```

Usage
=============
```bash
$ json2ssm --help
  usage: json2ssm [<flags>] <command> [<args> ...]
  
  Flags:
        --help     Show context-sensitive help (also try --help-long and
                   --help-man).
    -d, --debug    Enable debug logging.
        --version  Show application version.
  
  Commands:
    help [<command>...]
      Show help.
  
    put-json --json-file=JSON-FILE --encrypt [<flags>]
      Creates SSM parameters from the specified JSON file.
  
    get-json --path=PATH --decrypt
      Retrieves JSON document from SSM parameter store using given path (prefix).
  
    del-json --json-file=JSON-FILE
      Deletes parameters from SSM parameter store based on the specified JSON
      file.

```