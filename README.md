rigp
====

A restful geoip service.

How to build
============

1.  Install libGeoIP. With Homebrew, use the `geoip` formula. On Debian/Ubuntu,
    install `libgeoip-dev`.

2.  `go get github.com/codahale/geoip`

3.  In the base directory of the project: `go get && go build`

How to run
==========

Make sure you have `GeoLiteCity.dat` or `GeoIPCity.dat`, `GeoIPISP.dat` and
`GeoIPNetSpeed.dat` in your data directory.

1.  `./rgip -datadir /usr/local/share/GeoIP`

2.  Endpoints will be available for `/lookup/<ip>` and `/lookups/<ip>,<ip>,...`

Example: 

`http://localhost:8080/lookup/16.0.0.0`

```
{
  "ip": "16.0.0.0",
  "city": {
    "city": "Palo Alto",
    "country_code": "us",
    "latitude": 37.3762,
    "longitude": -122.1826,
    "region": "CA",
    "region_name": "California",
    "postal_code": "94304",
    "area_code": 650,
    "time_zone": "America/Los_Angeles"
  },
  "isp": "",
  "netspeed": "Unknown",
  "ufi": {
    "guessed_ufi": 0
  },
  "geohash": "9q9hesjk5h",
  "olc": "849V9RG8+FX"
}
```


For multiple IPs, use `http://localhost:8080/lookups/16.0.0.0,18.0.0.0`

Documentation
=============

For documentation, check [godoc](https://godoc.org/github.com/dgryski/rgip).
