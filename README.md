# Fukumu
Is a prototype of a container service docker like

## How to get started
### Clone the project
```bash
git clone https://github.com/FelipeMCassiano/Fukumu
```
### Run with root permissions (required)
```bash
sudo go run main.go <COMMAND>
```
### Commands
|Commands| Description
|---|---|
|`run`| run the program with a specific directory e.g. /bin/bash|
|`clean`| remove all files created in cgrops (can be useful if the process is been killed)| 

### Configure containers
> **_NOTE:_** Actually only one container is read by configuration file. I'm working foward to add more containers running simultaneously

fukumu.toml
```toml 
[[containers]]
memory = '2MB'
```
