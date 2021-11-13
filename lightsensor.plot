# Plots lightsensor.data file
#
# Usage:
# gnuplot lightsensor.plot

reset

set xdata time
set format x "%H"
set grid xtics

set term png size 600, 400
set output "lightsensor.png"

set xlabel "Time" # unixtimestamp in milliseconds
set ylabel "uLux"

plot "lightsensor.data" using (($1/1000)+(-4*3600)):2 with linespoints pt 0 notitle
# plot "lightsensor.data" using (($1/1000)+(-4*3600)):2:(0.00000001) smooth acsplines with linespoints pt 0 notitle
