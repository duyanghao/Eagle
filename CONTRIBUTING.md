# How to contribute

Welcome to Eagle!

## Code of Conduct

Eagle follows the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).

# Email and chat

- Email: [Eagle-email](1294057873@qq.com)

## Getting started

- Fork the repository on GitHub
- Read the README.md for build instructions

## Reporting bugs and creating issues

Reporting [bugs](https://github.com/duyanghao/Eagle/issues) is one of the best ways to contribute.

## Contribution flow

This is a rough outline of what a contributor's workflow looks like:

- Create a topic branch from where to base the contribution. This is usually master.
- Make commits of logical units.
- Make sure commit messages are in the proper format (see below).
- Push changes in a topic branch to a personal fork of the repository.
- Submit a pull request to [Eagle](https://github.com/duyanghao/Eagle/pulls).
- The PR must receive a LGTM from two maintainers found in the MAINTAINERS file.

Thanks for contributing!

### Preparation

- ***FORK*** [Eagle](https://github.com/duyanghao/Eagle) repository. The `Fork` button is in the top right corner of [Eagle](https://github.com/duyanghao/Eagle) home page.
- ***CLONE*** repository. `git clone https://github.com/<yourname>/Eagle.git`
- ***SET REMOTE***.
```
git remote add upstream https://github.com/duyanghao/Eagle.git
git remote set-url --push upstream no-pushing
```

### Code style

The coding style suggested by the Golang community is used in Eagle. See the [style doc](https://github.com/golang/go/wiki/CodeReviewComments) for details.

### Format of the commit message

We follow a rough convention for commit messages that is designed to answer two
questions: what changed and why. The subject line should feature the what and
the body of the commit should describe the why.

```
seeder: add seeder that provides meta info of blob to EagleClient and acts as the first uploader

Seeder stores blobs as files on disk backed by pluggable storage (e.g. FileSystem, S3) and provides meta info of blob to EagleClient, acting as the first uploader

Fixes #10
```

The format can be described more formally as follows:

```
<package>: <what changed>
<BLANK LINE>
<why this change was made>
<BLANK LINE>
<footer>
```

### Pull request across multiple files and packages

If multiple files in a package are changed in a pull request for example:

```
seeder/main.go
proxy/routes/route.go
```

At the end of the review process if multiple commits exist for a single package they
should be squashed/rebased into a single commit before being merged.

```
seeder: <what changed>
[..]
```

If a pull request spans many packages these commits should be squashed/rebased into a single
commit using message with a more generic `*:` prefix.

```
*: <what changed>
[..]
```