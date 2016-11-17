# P2PLive

## Introduction

A project to implements P2P live only use web-browser.

## Destination

To reduce load of many live stream server using Point to Point transport.

## Push Stream

ffmpeg -s 640x480 -f vfwcap -i 0 -r 30 -f mpeg1video -vf "crop=iw-mod(iw\,2):ih-mod(ih\,2)" -b 3000K -r 30 http://localhost:8080/stream

## Design

![Design](p2plive.png)

### CoOrdinate Server

Client request CoOrdinate Server for a tracker server, server returns a server based on the geographical position of Client and
 other infomation(eg. the client is in a LAN, and there's a tracker server specially for this LAN, eg. Zhejiang University)

The live stream is push to CoOrdinate Server in this version, later vesion will support multi-source live steam. The CoOrdinate Server
 will loop request Trackers to get all forward server and push live stream to them.

### Tracker Server

The Client will be orgnized by the tracker server. The tracker server keep a balance that most client get stream from 2 other client
 and push stream to 2 other client too. If this region of this tracker has less client, tracker tell them to directly pull from the
 forward server so that the forward server's load will be less.

Beginning

![Beginning](0.png)

A client close

![A client close](1.png)

Many transport net break off

![Many transport net break off](2.png)

Auto fix net

![Auto fix net](3.png)

### Forward Server

The Forward Server pull stream from CoOrdinate Server and push to client, keep connection with tracker to help tracker work.