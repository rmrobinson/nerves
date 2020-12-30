# googlerelayd

The Google Relay Daemon implements the server interface of the googlehome proto. It is designed to act as a command relay between the Google Smart Home Action API and multiple registered clients. It uses the agent ID provided at time of registration to route incoming Google Smart Home Action calls to the appropriate endpoint.

This will sit 'in the cloud' and be the central point for multiple registered agents.
