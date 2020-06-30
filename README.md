# gf
GoFingerprint (GF) helps quickly indentify web servers by checking their HTTP responses against a user defined list of fingerprints. Whether it's trying to determine which servers in your recon set are bootspring or testing for a specific response from a payload, gf is the tool for you!


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
