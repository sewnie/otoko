# otoko (音庫)

Simple CLI [bandcamp](https://bandcamp.com) collection synchronizer.

```
$ ./otoko -o ~/music
sjalvmord 7 / 71                                ⠼                                 
a2594097446 475.8 MiB / 518.4 MiB  █████████████████████████████████████████████████▌░░░░░ 2.30MiB/s
a382191878  466.5 MiB / 1.1 GiB    ██████████████████████▌░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 2.28MiB/s
a1737495501 266.3 MiB / 498.6 MiB  ████████████████████████████▌░░░░░░░░░░░░░░░░░░░░░░░░░░ 2.24MiB/s
a2783120099 75.5 MiB / 267.6 MiB   ███████████████▌░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 2.41MiB/s
t1433347432 29.3 MiB / 102.7 MiB   ███████████████▌░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 2.23MiB/s
a3457013254 18.5 MiB / 868.1 MiB   ▌░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░ 2.09MiB/s
```

## Installation
```sh
go install github.com/sewnie/otoko@latest
```

## Usage
`otoko` requires the `Cookie` header made on bandcamp web requests.

For convenience, the system browser cookies will be used if available, otherwise
`otoko` will fall back to reading from a cookie file.

The cookie file parameter (default `otoko-cookies.txt`) should be the value of
the `Cookie` header from a [bandcamp.com](https://bandcamp.com/) network request,
which can be found in a network request in the 'Request Headers' section under the network
requests tab in the browser.

Example usage:
```
otoko -format flac -o Music
```

## Behavior
Music is downloaded to the given output directory (`-o`, default `.`) with this structure:

```
Music
├── Sadness
│   └── atna
│       ├── 01 daydreaming.flac
│       ├── 02 how bright you shine.flac
│       ├── 03 hope you never forget.flac
│       └── cover.jpg
└── home is in your arms
    ├── _ (1433347432).flac
    └── _ (1987275855)
        ├── 01 _.flac
        ├── 02 _.flac
        └── cover.jpg
```
Music belonging to albums or tracks are saved without the artist and album in the
filename, since said metadata is already represented in the directory structure.

Albums and tracks with the same name will have their tralbum ID appended to the name.

If the track or album already exists, it is skipped.
