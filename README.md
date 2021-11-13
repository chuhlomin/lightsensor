# lightsensor

Attempt to plot light sensor data from [lunarsensor](https://lunar.fyi/sensor).

Buy the components, install firmware on Ambient Light Sensor.

Run the go app that polls data from `lunarsensor.local` host in your local network (for as long as you want to collect the data).

```bash
go run . | tee lightsensor.data
```
Plot the data with [gnuplot](http://www.gnuplot.info):

```bash
gnuplot lightsensor.plot
```

Open generated lightsensor.png

Automatically update lightsensor.png as you change lightsensor.plot:

```bash
fswatch lightsensor.plot | xargs -n1 sh -c "gnuplot lightsensor.plot"
```
