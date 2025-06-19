# Contributing to Collibra

:tada::thumbsup: First off all, thanks for taking the time to contribute to this Collibra repository!  :thumbsup::tada:

This page contains a set of guidelines to help you get started quickly and to make sure we, at Collibra, can keep things at the highest quality.

## Code of Conduct
To report an issue with this repository, we will be glad to help you through the usual channels, such as [Collibra Support](https://support.collibra.com/) or your Collibra representative. To directly submit some improvements to the code, we welcome any pull request and we will review it as soon as we can. Note, however, that by using or accessing the Developer Toolkit, you agree to the [Collibra Developer Terms](https://www.collibra.com/developer-terms), including section 11, which states: _"Collibra has no obligation to provide any maintenance or support for the Developer Toolkit (or to end users of your Add-Ons) or to fix any errors or defects"_.

## Prerequisites
- Install the correct version of Go (see go.mod for the version being used now)
- After you check out this git repository, make sure to execute the following command (in the root of the repository) to register the pre-commit hooks: `git config core.hooksPath .githooks`
- We use `golangci-lint` to check the code for quality. Please make sure to [install it](https://golangci-lint.run/usage/install/#local-installation)
