Jenkins Client Launcher [![Build Status](https://travis-ci.org/jkellerer/jenkins-client-launcher.png?branch=master)](https://travis-ci.org/jkellerer/jenkins-client-launcher)
=======================

_Jenkins Client Launcher_ is a simple native CLI executable that may be used to **bootstrap the Jenkins client** 
for connecting a computer with Jenkins (a.k.a Jenkins Node).

The main purpose of this project is to provide a tool that allows running Jenkins Nodes with 
**zero maintenance** in **user mode** having _Windows_ as the primary platform to support.

Features
--------

- Bootstrap the Jenkins client and keep it running.
- Autostart with the OS.
- Config file driven with support for centralized configuration.
- Prepare & maintain the environment:
	- Keep Jenkins client up-to-date.
	- Install Java if required.
	- Clean temporary folders.
- Monitoring with restart on failure:
	- Detect client crash e.g. OutOfMemory.
	- Detect remote offline state of the node by querying Jenkins server.

Limitations / Roadmap / Wishlist
--------------------------------

- More tests
- SSH tunnel support (connect to Jenkins over SSH)
- SSH server mode (allow connections from Jenkins)
- Register new nodes automatically (using the Jenkins API)


Usage
-----

###Displaying CLI Help

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
> launcher -help
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

_Hint:_ Help shows commandline options and it also creates a default config `launcher.config` 
inside the current working directory should it be missing.

###Attaching a node to Jenkins

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
> launcher -url=http://ci.tl/jenkins -secret=40cec6b0f...b9119008b07
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

###Attaching a node to Jenkins using centralized configuration

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
> launcher -defaultConfig=http://ci.tl/launcher.config 
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

**With `http://ci.tl/launcher.config` being**: 

```xml
<config runMode="client" autostart="false">
    <ci>
        <url>http://ci.tl/jenkins</url>
        <noCertificateCheck>false</noCertificateCheck>
        <auth>
            <user>admin</user>
            <password>changeit</password>
        </auth>
    </ci>
</config>
```

_Hint:_ Add `-name=name-of-node-in-jenkins` if the node name is not the same as the hostname of 
the computer where the launcher is started.

###Autostart next time the OS boots

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
> launcher -autostart=true
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

_Windows Hint:_ You should setup _Windows_ to boot directly to the desktop without asking for a password 
in order to let autostart run the launcher. 

###Note on Jenkins Setup

The currently implemented run mode **client** requires the manual creation of a _Dumb Slave_ in Jenkins using the following settings:

Parameter             | Value
----------------------|-----------------------------
Name                  | **name-of-your-node** _(preferably the `hostname` of the computer as otherwise it has to be specified with `launcher -name=some-other-name`)_
Launch method         | **Launch slave agents via Java Web Start**

Building
--------

- **Go >= 1.2** is required
- **Create Go Workspace** if missing:

```Batchfile
mkdir go-workspace
cd go-workspace 
set "GOPATH=%CD%" REM Add this to the environment if you plan to do more than just one build.
```

- **Download & Build**:

```Batchfile
go get github.com/jkellerer/jenkins-client-launcher
cd src\github.com\jkellerer\jenkins-client-launcher

go get ...
go build launcher.go
```


Developing
----------

- A Go IDE is recommended (e.g. one of IntelliJ, LiteIDE, Eclipse)
- The project ROOT is `go-workspace\src\github.com\jkellerer\jenkins-client-launcher`.\
  `go` should be called from this path.
- **Installing Dependencies**:

```Batchfile
go get ./...
```

- **Building**:

```Batchfile
go build launcher.go
```

- **Running All Tests**:

```Batchfile
go test ./...
```


High Level Architecture
-----------------------

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
                        +-------------------------+     +--------------+
                        | launcher.Run() {...}    | <-- | Config (xml) |
                        +-------------------------+     +--------------+
              Execute 1st  /              \
    +-------------+       /                \  Execute 2nd
    | environment |      /              +-------+
    +-------------+--------------+      | modes |
    | EnvironmentPreparer:       |      +-------+-----------------------------+
    | - JavaDownloader/Installer |      | <Loop... until term signal>         |
    | - JenkinsNodeMonitor       |      +-------------------------------------+
    | - JenkinsClientDownloader  |      |         [Active RunMode?]           |
    | - TempLocationCleaner      |      |             /     \                 |
    | - ...                      |      |  +----------+     +------------+    |
    +----------------------------+      |  | Client   |     | SSH server |    |
                                        |  +----------+     +------------+    |
                                        |                                     |
                                        +-------------------------------------+
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~


License
-------

MIT, see [LICENSE](LICENSE)
