NSE 
============================

Network Service Endpoint (NSE) binary in this folder is a simple example of NSE 
component of Network Service Mesh.
It used for CI integration testing purposes but it can also be used as an example
of a API calls flow for real NSE applications.

Overview
------------
NSE statically defines a channel, "Channel-1" and then advertises it to NSM. In 
"Channel advertisement", nse also send information for a listening socket. This
socket is used by NSM to connect to NSE in case of Connection Request call.

Start
------------
Folder "./conf/sample" contains "nse.yaml" file which can be used to start nse process.