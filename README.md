# External Issue Sources

This library is a fast, efficient app that scrapes and returns structured data from an external issue source, such as `comics.org` or `comicbookdb.com`.

This library is private to hide how the issue sources are fetched from the main application.

## Getting Started

1. Run `make up` to create the docker container to run the application.
2. Run `make install-deps` to install the vendor dependencies.

## Useful commands
- `make test` - Run the applicaiton tests.
- `make format` - Format the go files.

## Application info

### models.go
Defines the objects that are returned from the parsers.

### parsers.go
The parsers are responsible for taking in an `io.body` and reading data to parse issue information from an external source.

### sources.go
Fetches objects, such as a character or a character search result page, via an HTTP call and 

Some some comic book characters have thousands of issues, so `Character(url string) (*Character, error)` concurrently gathers as many issues as configured and returns the character with all its issues attached.
