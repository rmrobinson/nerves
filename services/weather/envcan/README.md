# envcan

This package provides a service for interfacing with the Environment Canad Weather Office service. It needs to be bootstrapped by running the `getstations` tool to initialize the list of locations and URLs to retrieve weather data from; once setup the envcan.Service implementation can be used as a weather service for Canadian locations.
