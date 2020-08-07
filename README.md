# gofingerprint
GoFingerprint helps quickly indentify web servers by checking their HTTP responses against a user defined list of fingerprints. Whether it's trying to determine which servers in your recon set are bootspring or testing for a specific response from a payload, gf is the tool for you!

Thanks to @nahamsec https://github.com/nahamsec for the tool idea!

### usage options
```
  -badpath string
        The intentional 404 path to hit each target with to get a response. (default "/sfdrbdbdb")
  -body string
        Data to send in the request body
  -debug
        Enable to see any errors with fetching targets
  -fingerprints string
        JSON file containing fingerprints to search for.
  -method string
        which HTTP request to make the request with. (default "GET")
  -output string
        Directory to output files (default "./")
  -timeout int
        timeout for connecting to servers (default 10)
  -workers int
        Number of workers to process urls (default 20)

```

### basic usage

```cat targets | gofingerprint -fingerprints ./fingerprints.json```


### fingerprint file format (example can be found in fingerprints directory)

```
[
  {
    "name": "<UNIQUE NAME OF FINGERPRINT>",
    "fingerprint" : ["<SEARCH TEXT USED TO ID SERVICE OR PRODUCT>"] #allows multiple fingerprints
  }
]
```
