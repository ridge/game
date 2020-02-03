# Contributing

Contributions are more than welcome. Please read these guidelines for making the
process as pleasant as possible.

## Discussion

Development discussion is taking place on the `#game` channel on [gophers
slack](https://gophers.slack.com/).

## Issues

If there's an issue you'd like to work on, please comment on it, so we can
discuss the approach and avoid duplicating the work.

Please always create an issue before sending a PR unless it's an obvious typo or
other trivial change.

## Dependency Management

Currently Game has no dependencies outside the standard libary. Let's keep it
that way.

Adding dependencies to Game increases probability of version conflicts for every
project that uses Game. In a perfect world there would be no problems using
different versions of the same library by Game and by the project, but in
reality there are libraries that break backward compatibility without bumping
their major version.

The exception to this rule is testing libraries. If a library is used
exclusively in `*_test.go` files it does not contribute to transitive
dependencies of Game.

## Versions

Game works with Go >= 1.13. CI checks it.

## Testing

Please write tests for new features and bugfixes.

These tests ought use the go `testing` package and pass race detector (`go test
-race ./...`).
