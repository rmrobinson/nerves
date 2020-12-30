# googlebridged

The Google Bridge Daemon implements the client interface of the googlehome proto. It is designed to act as a translator between the Google Smart Home Action framework and the nerves domotics framework. A given instance of the googlebridged registers with a pre-specified agent ID (corresponding to whichever Google account is linked to this home) and then translates the Google Smart Home commands into domotics bridge commands. It also listens for updates from the domotics system and propogates these back to the Google framework.

It is intended that any sort of domotics -> Smart Home Action translation happens in this framework. Whatever is sent to the server interface of the googlehome proto should have no domotics-specific concepts in it; likewise it is the responsibility of this program to translate the incoming updates from the Google Smart Home Action framework into corresponding domotics terms.
