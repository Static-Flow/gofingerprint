# gf
GoFingerprint (GF) is a Go tool for taking a list of target webservers and checking their HTTP response against a user defined list of fingerprints to try and determine what product or service is running on a target.


### usage options
```
  -path string
        The path to hit each target with to get a response.
  -debug
        Enable to see any errors with fetching targets
  -fingerprints string
        JSON file containing fingerprints to search for.
  -output string
        Directory to output files (default "./")
  -timeout int
        timeout for connecting to servers (default 10)
  -workers int
        Number of workers to process urls (default 20)
```

### basic usage

```cat targets | gf -path "/" -fingerprints ./fingerprints.json```


### fingerprint file format (example can be found in fingerprints directory)

```
[
  {
    "name": "<UNIQUE NAME OF FINGERPRINT>",
    "fingerprint" : "<SEARCH TEXT USED TO ID SERVICE OR PRODUCT>"
  }
]
```
