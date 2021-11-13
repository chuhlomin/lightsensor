# lightsensor

Attempt to plot light sensor data from [lunarsensor](https://lunar.fyi/sensor).

Buy the components, install firmware on Ambient Light Sensor.

Build the go app that polls data from `lunarsensor.local` host in your local network:

```bash
go build -o lightsensor main.go
```

Run the app for as long as you want to collect the data:

```bash
./lightsensor | tee lightsensor.data
```

Plot the data with [gnuplot](http://www.gnuplot.info):

```bash
gnuplot lightsensor.plot
```

Open generated lightsensor.png

![lightsensor](https://user-images.githubusercontent.com/3620471/141604928-b5731606-a28f-4e16-ad55-f0302af2047d.png)

Automatically update lightsensor.png as you change lightsensor.plot:

```bash
fswatch lightsensor.plot | xargs -n1 sh -c "gnuplot lightsensor.plot"
```
