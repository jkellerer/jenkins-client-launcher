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
- Register nodes in Jenkins if missing.
- Config file driven with support for centralized configuration.
- Prepare & maintain the environment:
	- Keep Jenkins client up-to-date.
	- Check if Java installation is required.
	- Remove outdated temporary files and workspaces.
- SSH tunnel support (connect to Jenkins via SSH).
- Monitoring with restart on failure:
	- Detect client crash e.g. OutOfMemory.
	- Detect remote offline state of the node by querying Jenkins server.

Limitations / Roadmap / Wishlist
--------------------------------

- More tests.
- File based logging.
- Install Java when required.
- SSH server mode (allow connections from Jenkins).


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

###Attaching a node to Jenkins using centralized configuration (HTTP)

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

_Hints:_ 

- Add `-name=name-of-node-in-jenkins` if the node name is not the same as the hostname of 
  the computer where the launcher is started:
  
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
> launcher -defaultConfig=http://ci.tl/launcher.config -name=name-of-node-in-jenkins  
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  
- Add `-create` if a new node should be created in Jenkins when no matching node can
  be found. When required the launcher creates a new JNLP type node using the computer's hostname
  as node name (by default, if not overridden):
  
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
> launcher -defaultConfig=http://ci.tl/launcher.config -create  
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  
###Attaching a node to Jenkins using centralized configuration (Network Share)

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
> \\share\launcher -defaultConfig=. -directory=C:\Jenkins-CI\  
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

**With `\\share\` being**:

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
\\share\
    launcher.exe
    launcher.config
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

_Note:_ `-defaultConfig=.` is an alias to `-defaultConfig=\\share\launcher.config` 

###Tunneling the JNLP client connection via SSH

Add the following section to `launcher.config`: 

```xml
<config ... >
    <ci>
        ...
        <tunnel>
            <jnlp>
                <ssh>
                    <enabled>true</enabled>
                    <address>ssh-host</address>
                    <port>22</port>
                    <fingerprint>65:30:38:96:35:56:4f:64:64:e8:e3:a4:7d:59:3e:19</fingerprint>
                    <auth>
                        <user>admin</user>
                        <password>changeit</password>
                    </auth>
                </ssh>
            </jnlp>
        </tunnel>
    </ci>
</config>
```

###Autostart next time the OS boots

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
> launcher -autostart=true
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

_Windows Hint:_ You should setup _Windows_ to boot directly to the desktop without asking for a password 
in order to let autostart run the launcher. 

###Note on Jenkins Setup

The run mode **client** requires the creation of a _Dumb Slave_ in Jenkins using the following settings:

Parameter             | Value
----------------------|-----------------------------
Name                  | **name-of-your-node** _(preferably the `hostname` of the computer as otherwise it has to be specified with `launcher -name=some-other-name`)_
Launch method         | **Launch slave agents via Java Web Start** (JNLP)

_Hint:_ Adding `-create` to the commandline enables auto creation of the node inside Jenkins, should it be missing.

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
- The project ROOT is `go-workspace\src\github.com\jkellerer\jenkins-client-launcher`.<br/>
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
    | - LocationCleaner          |      |             /     \                 |
    | - ...                      |      |  +----------+     +------------+    |
    +----------------------------+      |  | Client   |     | SSH server |    |
                                        |  +----------+     +------------+    |
                                        |                                     |
                                        +-------------------------------------+
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~


License
-------

MIT, see [LICENSE](LICENSE)
