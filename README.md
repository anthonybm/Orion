# Orion
 a framework for triage of relevant incident response and forensics artifacts from various operating systems

[![Latest version](https://img.shields.io/badge/version-v0.2.0-blue)](https://github.com/tonythetiger06/goMass/releases/tag/v0.2.0-alpha)
[![status](https://img.shields.io/badge/status-alpha-red)]
[MIT](https://choosealicense.com/licenses/mit/)

### Usage 
This is an alpha - work in progress! Please read all documentation and review before running on your own system. Of note: 
- The configs/ folder contains a mac and windows config sample, all present keys are required
- The modules listed in each are what exist at this time, comments will denote WIP/experimental work

### Testing usage example
./Orion -m mac -f csv -o output -c configs/mac.toml -l debug -T

### Actual usage 
sudo ./Orion -m mac -f csv -o output -c path_to/mac.toml -l info

Orion currently has functionality to
 - Create and integrate modules for MacOS (many written) and Windows (one example file system walk written)
 - Log errors, debug, warning, and input statements
 - Output logs in JSON format
 - Output for modules in CSV format
 - Tested on OSX 10.15.5 and Windows 10

## Roadmap
 - Testing :) 
 - Ensure documentation is sufficient
 - Graceful exit on SIGINT
 - More modules for MacOS
 - Sign for MacOS? 
 - More modules for Windows
 - Support Linux module writing
 - Support no-logging mode
 - Support for module output in other formats than CSV
 - Support for tarballing module output
 - Support for uploading module output 

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change. 

Please make sure to create/update tests as appropriate.

## Module Development
### tl;dr; see example code
- Prior to writing code, determine where on the filesystem the artifact resides, how it is manually parsed, and what the artifact is used for
- Follow the gitflow methodology [here](https://www.atlassian.com/git/tutorials/comparing-workflows/gitflow-workflow) to create an appropriate Feature Branch following name convention of 'feature/**operating_system**/**feature_name**'
- Under the appropriate OS source folder, create a package for the module and in that package create a file for the module code following the convention of "modulename.go"
- You are required to have a "ModuleNameModule" struct type that implements a Start() function
- You are required to add the "ModuleNameModule" struct to the type registry in module.go
- You are required to add the module struct name to the config file
- Follow the SampleModule for conventions

## License 
This project is licensed under the terms of the MIT license. See LICENSE for details.