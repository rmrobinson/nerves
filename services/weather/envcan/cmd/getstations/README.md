# getstations

This is a small tool that retrieves the list of weather stations with RSS feeds that Environment Canada releases as part of the [Weather Office](https://weather.gc.ca/mainmenu/weather_menu_e.html) service. For each URL it discovers, it uses National Resources Canada (NRC)'s [geocoding API](https://www.nrcan.gc.ca/earth-sciences/geography/place-names/tools-applications/9249) to determine the latitude and longitude of the weather station.

This tool attempts to limit its load on these freely available services by making all requests serially, and isn't intended to be run frequently. Once the output file is generated it should need only infrequent updating.