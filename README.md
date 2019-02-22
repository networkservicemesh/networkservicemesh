# Network Service Mesh

[![CircleCI Build Status](https://circleci.com/gh/networkservicemesh/networkservicemesh/tree/master.svg?style=svg)](https://circleci.com/gh/networkservicemesh/networkservicemesh/tree/master)
[![Weekly minutes](https://img.shields.io/badge/Weekly%20Meeting%20Minutes-Tue%208am%20PT-blue.svg?style=plastic")](ttps://docs.google.com/document/d/1C9NKjo0PWNWypROEO9-Y6haw5h9Xmurvl14SXpciz2Y/edit#heading=h.rc9df0a6n3ng)
[![Mailing list](https://img.shields.io/badge/Mailing%20List-networkservicemesh-blue.svg?style=plastic")](https://groups.google.com/forum/#!forum/networkservicemesh)
[![GitHub license](https://img.shields.io/badge/license-Apache%20license%202.0-blue.svg)](https://github.com/networkservicemesh/networkservicemesh/blob/master/LICENSE)
[![IRC](https://www.irccloud.com/invite-svg?channel=%23networkservicemesh&amp;hostname=irc.freenode.net&amp;port=6697&amp;ssl=1)](http://webchat.freenode.net/?channels=networkservicemesh)

<p align="center">
  <a href="https://www.networkservicemesh.io/"><img src="https://networkservicemesh.io/img/logo.png"></a>
</p>

## What is Network Service Mesh

Network Service Mesh (NSM) is a novel approach solving complicated L2/L3 use cases in Kubernetes that are tricky to address with the existing Kubernetes Network Model. Inspired by Istio, Network Service Mesh maps the concept of a Service Mesh to L2/L3 payloads as part of an attempt to re-imagine NFV in a Cloud-native way!

For more information, have a look at our detailed overview of NSM - [What is NSM?](/docs/what-is-nsm.md)

## Getting started

To get started, follow our [Quick Start](/docs/guide-quickstart.md) guide.

If you want a more extensive look, you can go follow the slightly more detailed [Build](/docs/guide-build.md) page.

## Docs

See our full documentation at the [docs](/docs/README.md) folder.

## Get involved

* Weekly meetings
  * [Zoom meeting - General](https://zoom.us/my/networkservicemeshwg) - every Tuesday at 8:00 - 9:00 AM PT
  * [Zoom meeting - Documentation](https://zoom.us/my/networkservicemeshwg) - every Wednesday at 8:00 - 9:00 AM PT
  * [Meeting minutes - General](https://docs.google.com/document/d/1C9NKjo0PWNWypROEO9-Y6haw5h9Xmurvl14SXpciz2Y/edit#heading=h.rc9df0a6n3ng)
  * [Meeting minutes - Documentation](https://docs.google.com/document/d/1113nzdL-DcDAWT3963IsS9LeekgXLTgGebxPO7ZnJaA/edit?usp=sharing)
  * [Calendar](https://calendar.google.com/calendar/embed?src=iae5pl3qbf2g5ehm6jb2h7gv08%40group.calendar.google.com&ctz=America%2FLos_Angeles)
  * [Recordings](https://www.youtube.com/playlist?list=PLj6h78yzYM2Ob01WuF-mqMxd8a_wAuppb)
  * [Google Drive Folder](https://drive.google.com/drive/folders/1f5fek-PLvoycMTCp6c-Dn_d9_sBNTfag) - where most of the NSM docs are located. Keep in mind that some of them are still Work in Progress.

* Channels
  * [![IRC](https://www.irccloud.com/invite-svg?channel=%23networkservicemesh&amp;hostname=irc.freenode.net&amp;port=6697&amp;ssl=1)](http://webchat.freenode.net/?channels=networkservicemesh)

* Mail
  * [![Mailing list](https://img.shields.io/badge/Mailing%20List-networkservicemesh-blue.svg?style=plastic")](https://groups.google.com/forum/#!forum/networkservicemesh)

## References

* [Official Website](https://www.networkservicemesh.io/)
* [Network Service Mesh: A Narrative Introduction](https://docs.google.com/presentation/d/1IC2kLnQGDz1hbeO0rD7Y82O_4NwzgIoGgm0oOXyaQ9Y/edit#slide=id.p) <- Start Here
  * [Sarah and the secure-intranet-connectivity](https://docs.google.com/presentation/d/1IC2kLnQGDz1hbeO0rD7Y82O_4NwzgIoGgm0oOXyaQ9Y/edit#slide=id.g4042dbe7c0_11_7)
     * [Presented to Kubernetes Resource Management Working Group](https://www.youtube.com/watch?v=pToCYxp5Kgs)
  * [Hannah and the Hardware NICs](https://docs.google.com/presentation/d/1IC2kLnQGDz1hbeO0rD7Y82O_4NwzgIoGgm0oOXyaQ9Y/edit#slide=id.g4042dbe7c0_11_16)
* [Network Service Mesh: Intro slides](https://docs.google.com/presentation/d/1C3r91ev0tWnFFUjiV4W84Hp965YGR1D9lChZo73Jwq0/edit#slide=id.g375263091c_1_0)
* [Network Service Mesh: Intro slides presented](https://www.youtube.com/watch?v=f2FV6C_dSk4)
* [Network Service Mesh: HW Interfaces](https://drive.google.com/open?id=1_nwt1tTy-RWYHDj70-2g6g7OvBuuyGpCbyEREjdZkNU)
* [Network Service Mesh: Distributed CNFs(Distributed Bridges)](https://drive.google.com/open?id=1j78oj_5bJ23dydFT-FTrMwlSrMkHPGC70qmjQzQRPJ4)
* [Network Service Mesh: VPN Gateway](https://docs.google.com/presentation/d/1BnouS8d_Aesq9IPRPWRxTcZR1ZtmULcyh6l0gAK204Q/edit#slide=id.p)
* [Network Service Mesh: Presentation to Kubernetes SIG Networking 2018-05-31](https://docs.google.com/presentation/d/1vmN5EevNccel6Wt8KgmkXhAfnjIli4IbjskezQjyfUE/edit#slide=id.p)
* [Use Case Working Document](https://drive.google.com/open?id=1bIK_SF8lnP1IrZQUIj4eAuDyibSI6tpMvE_bF3RKSCk)

## Dependencies

The Network Service Mesh project uses [Dependabot](https://dependabot.com/) to manage dependencies.
Dependabot pushes out dependency updates to ensure our dependencies remain current and we're not stuck
with code with known vulnerabilities.

## FAQ

If you run into problems, check the [docs](/docs/README.md) and feel free to post issues in the [Network Service Mesh](https://github.com/networkservicemesh/networkservicemesh/issues) repository.

## Licence

This project is released under the Apache 2.0 License. Please review the [License file](LICENSE) for more details.
