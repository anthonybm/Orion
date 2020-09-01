# Contributing to Orion

🙏:tada: Thank you for being interested in contributing! :tada:🙏

Orion is an open source project and contributions from the community will always be considered! There are many ways to contribute, from writing tutorials or blog posts, improving the documentation, submitting bug reports and feature requests or writing code which can be incorporated into the framework! 

## Table Of Contents

* [Code of Conduct](#code-of-conduct)
* [Before Getting Started](#before-getting-started)
* [Why Should I Contribute?](#why-should-i-contribute)
* [How Can I Contribute?](#how-can-i-contribute)
* [Where do I Start?](#where-do-i-start)
* [Styleguides](#styleguides)

## Code of Conduct
This project and everyone participating in it is governed by the [Code of Conduct](https://github.com/anthonybm/Orion/blob/master/CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Before Getting Started
Orion is an artifact triage tool framework that facilitates independent modules to run and collect whatever forensic artifact is specified. Orion allows contributers to add functionality to parse and triage unimplemented forensic artifacts in the form of "modules", which essentially are packages in Go that contain the required constructs and supplementary functionality to parse the artifacts. 

This ease of adding functionality in the form of a single file and few lines of change was inspired by other existing open source tools already written in Python. The benefit of Orion over others is that it compiles down to a single binary that utilizes a config file, rather than requiring various source and third-party code to be copied when you want to utilize the tool. 

### Conventions
* Preferred editor is [Visual Studio Code](https://code.visualstudio.com/)
* Orion follows the [Gitflow Workflow methodology](https://www.atlassian.com/git/tutorials/comparing-workflows/gitflow-workflow). Check it out :smile:
* Apply [build constraints](https://golang.org/cmd/go/#hdr-Build_constraints) where required 

## Why Should I Contribute?
* You think this project is awesome but could be something more!
* You want to help the DFIR community!
* You think something is being done wrong!

## How Can I Contribute?
* Report bugs
* Suggest Features
* Create or update documentation
* Create or update a module

## Where do I Start?
The following sections describe how you actually go about contributing :) 
### Pre-requisites

* [Go version 1.14](https://golang.org/dl/) installed and configured
* C compiler for cgo modules [such as this for Windows](https://jmeubank.github.io/tdm-gcc/download/)
* Access to macOS system for writing Mac modules or building Mac binary
* Access to Windows system for writing Windows modules or building Windows binary
* Have a feature in mind either from the existing issues list or something entirely new! Make sure you know what the artifact(s) is used for, where it is located on the filesystem, and document anything else as you see fit. It will be helpful to look at the feature change template to see what information is suggested to have.
* **All of your suggested changes should go into a specific `feature branch`, which should be based off of the `develop` branch**

1) Fork the Orion repository. Follow the [GitHub Help instructions](https://help.github.com/articles/fork-a-repo/) on how to fork a repo if you're not sure how.
2) Clone it to your local machine and navigate to the directory where you've cloned the source code
3) Make sure you sync and [fetch all remote branches](https://www.atlassian.com/git/tutorials/syncing/git-fetch) (or just the `develop` branch)
3) You should create a feature branch based off of the `develop` branch following the name convention of `feature_windows_modulenamehere` (where you replace windows with the operating system you are writing for and replace modulenamehere with a unique module name).
4) See one of the sections below for specific contribution instructions

### Writing a new module
see example code :wink:
#### What do I need to touch?
To add a new module to Orion, you will need to touch/create code in the following places: 
* the os specific folder (i.e. `windows/` or `mac/` )
* the `util/` folder if you are adding a general purpose utility (i.e. a Chrome timestamp converter function, common structure traversal function)
#### How do I go about this?
0) Prior to writing code, determine where on the filesystem the artifact resides, how it is manually parsed, and what the artifact is used for
1) Follow the gitflow methodology [here](https://www.atlassian.com/git/tutorials/comparing-workflows/gitflow-workflow) to create an appropriate Feature Branch following name convention of 'feature/**operating_system**/**feature_name**'
2) Under the appropriate OS source folder (i.e. `windows/` or `mac/` ), create a package for the module and in that package create a file for the module code following the convention of "modulename.go"
3) You are required to have a "ModuleNameModule" struct type that implements a Start() function
```
// +build darwin

package macsample
// ... omitted supporting code
type MacSampleModule struct {
}

func (m MacSampleModule) Start(inst instance.Instance) error {
	err := m.osVersion(inst)
	if err != nil {
		zap.L().Error("Error running MacSampleModule: " + err.Error())
	}
	return err
}
// ... omitted supporting code
```
4) You are required to add the "ModuleNameModule" struct to the type registry in `engine/module_*.go`
```
func init() {
	registerType((*macsample.MacSampleModule)(nil))
   // ...
}
```
5) You are required to add the module struct name to the config file if you want it to execute when you run Orion
```
modules = [ # Comment out what you do not need 
   "MacSampleModule",
   // ... omitted other module lines
]
```
6) Convention suggests that your module produce output! 
Your module's output will be written to a file `orionRuntime + "_" + module + "." + outputtype`.
```
// This line should be at the start of your module's entry function (in this example, the func (m MacSampleModule) osVersion(inst instance.Instance))
dw, err := datawriter.NewOrionWriter(moduleName, inst.GetOrionRuntime(), inst.GetOrionOutputFormat(), inst.GetOrionOutputFilepath())
... omitted module functionality
// Write to output :) (error checks omitted for brevity) 
err = mw.WriteHeader(header)
err = mw.WriteAll(values)
err = mw.Close()
```

7) See the SampleModule for other conventions

## Styleguides
### Git Commit Messages

* Use the present tense ("Add feature" not "Added feature")
* Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
* Limit the first line to 72 characters or less
* Reference issues and pull requests liberally
* When only changing documentation, include `[ci skip]` in the commit description
* Consider starting the commit message with an applicable emoji:
    * :art: `:art:` when improving the format/structure of the code
    * :cake: `:cake:` when edit templates or/and css styles
    * :rocket: `:rocket:` when improving performance
    * :memo: `:memo:` when writing docs
    * :bug: `:bug:` when fixing a bug
    * :fire: `:fire:` when removing code or files
    * :green_heart: `:green_heart:` when fixing the CI build
    * :white_check_mark: `:white_check_mark:` when adding tests
    * :lock: `:lock:` when dealing with security
    * :arrow_up: `:arrow_up:` when upgrading dependencies
    * :arrow_down: `:arrow_down:` when downgrading dependencies
    * :dolphin: `:dolphin:` when add new migrations
    * :shirt: `:shirt:` when removing linter warnings
    * :watermelon: `:watermelon:` when you add or edit translations.
    * :gem: `:gem:` when you creating new release
    * :bookmark: `:bookmark:` when creating new tag
    * :ambulance: `:ambulance:` when you adding critical hotfix
    
### Branches
* Latest Release - master
* Upcoming Release - bug fixes and documentation -> release/<ver_no>
* Feature - branches from develop -> feature/<opsys>_<module/feature>
