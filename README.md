# HL7 v2 MLLP to Aidbox proxy

[![Build
Status](https://travis-ci.org/HealthSamurai/hl7proxy.svg?branch=master)](https://travis-ci.org/HealthSamurai/hl7proxy)

This tiny utility is intended to capture HL7 v2 traffic and submit
every received message to the Aidbox FHIR Server where message parsing
and mapping will happen.

Most likely you'll want to deploy this utility to some host inside
your organization's infrastructure to avoid setting up VPN or any
other tunneling to external network. `hl7proxy` is self-sufficient
binary which means it doesn't have any dependencies (DLLs, packages,
etc). All you need is to pick a binary for your OS/architecture, drop
to target host and then run it.

Precompiled binaries are available on
[Releases](https://github.com/HealthSamurai/hl7proxy/releases)
page. For now there is only one [Latest Unstable
Build](https://github.com/HealthSamurai/hl7proxy/releases/tag/edge).

## Usage

`hl7proxy` is a command-line utility, so you need to start Terminal or
Console application for your operating system. Also on Linux/OSX
you'll need to make binary executable after downloading it:

```
chmod +x ~/Downloads/hl7proxy-linux-amd64
```

(don't forget to alter the filename if you use different
OS/architecture)

`cd` into the directory where `hl7proxy` file is located and invoke
it:

```
./hl7proxy -port 5000
```

If you see a message `Listening to :5000` then `hl7proxy` is running
and ready to accept connections.

## Running as a Windows service

To be written. Short answer: use [NSSM](http://nssm.cc/) to register
`hl7proxy` as a Service.


## Support

Feel free to join "Aidbox Users" Telegram group to ask questions:
https://t.me/aidbox

## License

Copyright © 2019 [Health Samurai](https://health-samurai.io/) team.

hl7proxy is released under the terms of the MIT License.
