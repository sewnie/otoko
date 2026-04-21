# otoko (音庫)

Simple CLI [bandcamp](https://bandcamp.com) collection synchronizer and manager.

```
$ ./otoko --format flac ~/music
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

```
Usage: otoko <command> [flags]

Flags:
  --identity=    Bandcamp identity cookie value, fetched from browser if empty ($BANDCAMP_IDENTITY)

Commands:
  value [flags]
    Calculate the total value of your Bandcamp collection

  sync <output> [<tralbums> ...] [flags]
    Download and synchronize your collection to a local directory

  list [flags]
    Display detailed metadata for tracks and albums in your collection
```

`otoko` requires the `identity` cookie for making authorized requests on Bandcamp.


The identity parameter `--identity` must be the value of the `identity` cookie, found from
the `Cookie` header in a [bandcamp.com](https://bandcamp.com/) network request,
which can be found in a network request in the 'Request Headers' section under the network
requests tab in the browser.

## Behavior
Music is downloaded to the given output directory with this structure:

```
[Output]
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

Unfortunately, the directory structure will keep the directory name of collaboration
albums equal to that of the "artist" field in the tralbum data. This is due to the
fact that Bandcamp has no way of storing a list of artists, and artists use the
single artist name to represent multiple, either using commas or slashes. Due to the
fragility of this naming scheme, the music player using this directory structure should
attempt to read the music tags or the user to use MusicBrainz Picard.

Albums and tracks with the same name will have their tralbum ID appended to the name,
unless they are downloaded seperately, in which case this check will be ignored.

If the track already exists or the album's tracklist matches, it is skipped. The
option `--strict` will add checking the downloaded format as well.
