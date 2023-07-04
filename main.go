package main

import (
	"fmt"
	"github.com/gizak/termui/v3/widgets"
	"math/rand"
	"net"
	"os"
	"time"

	ui "github.com/gizak/termui/v3"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

var data []float64
var plot *widgets.SparklineGroup
var spark *widgets.Sparkline

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: ping <hostname>")
		os.Exit(1)
	}
	host := os.Args[1]
	draw(host)
	//traceroute.Traceroute(host)
}

func draw(host string) {
	if err := ui.Init(); err != nil {
		panic(err)
	}
	defer ui.Close()

	// 创建一个折线图 widget

	spark = widgets.NewSparkline()
	spark.Title = "Sparkline (ms)"
	spark.LineColor = ui.ColorGreen

	plot = widgets.NewSparklineGroup(spark)

	// get new window size
	var height, width int
	width, height = ui.TerminalDimensions()
	plot.Title = "Go Ping Graph"
	// update sparkline group size
	plot.SetRect(0, 0, width, height-3)
	// rerender
	ui.Render(plot)
	// 开始定时器添加新的数据
	go ping(host)
	// 处理退出事件
	for e := range ui.PollEvents() {
		if e.Type == ui.KeyboardEvent {
			break
		}
		if e.Type == ui.ResizeEvent {
			// get new window size
			width, height = ui.TerminalDimensions()
			// rerender
			// update sparkline group size and x axis title
			plot.SetRect(0, 0, width, height-3)
			// rerender
			ui.Render(plot)
		}
	}
}

// movingAverage 平滑数据使用移动平均算法。
// data 是输入数据，windowSize 是移动平均的窗口大小。
func movingAverage(data []float64, windowSize int) []float64 {
	if windowSize <= 0 {
		panic("movingAverage: windowSize must be greater than 0")
	}

	result := make([]float64, len(data))

	// 使用双重循环来计算移动平均
	for i := 0; i < len(data); i++ {
		windowStart := i - windowSize/2
		if windowStart < 0 {
			windowStart = 0
		}

		windowEnd := i + windowSize/2
		if windowEnd > len(data) {
			windowEnd = len(data)
		}

		sum := 0.0
		for j := windowStart; j < windowEnd; j++ {
			sum += data[j]
		}

		result[i] = sum / float64(windowEnd-windowStart)
	}

	return result
}

func calculateStatistics(numbers []float64) (float64, float64, float64) {
	// 初始化最大值和最小值为第一个数字
	maximum := numbers[0]
	minimum := numbers[0]
	sum := 0.0

	// 遍历数组找到最大值和最小值，并计算总和
	for _, num := range numbers {
		if num > maximum {
			maximum = num
		}
		if num < minimum {
			minimum = num
		}
		sum += num
	}

	// 计算平均值
	average := sum / float64(len(numbers))

	return maximum, minimum, average
}

func ping(host string) {
	c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer c.Close()

	dst, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		// Make a new ICMP echo request packet
		b, err := (&icmp.Message{
			Type: ipv4.ICMPTypeEcho, Code: 0,
			Body: &icmp.Echo{
				ID: os.Getpid() & 0xffff, Seq: 1,
				Data: []byte("HELLO-R-U-THERE"),
			},
		}).Marshal(nil)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Send the packet
		startTime := time.Now()
		n, err := c.WriteTo(b, dst)
		if err != nil {
			fmt.Println(err)
			return
		} else if n != len(b) {
			fmt.Println("got short write from WriteTo")
			return
		}

		// Wait for a reply
		reply := make([]byte, 1500)
		err = c.SetReadDeadline(time.Now().Add(10 * time.Second))
		if err != nil {
			fmt.Println(err)
			return
		}
		n, _, err = c.ReadFrom(reply)
		//n, peer, err := c.ReadFrom(reply)
		if err != nil {
			fmt.Println(err)
			return
		}
		duration := time.Since(startTime)
		rm, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), reply[:n])
		if err != nil {
			fmt.Println(err)
			return
		}
		switch rm.Type {
		case ipv4.ICMPTypeEchoReply:
			//fmt.Printf("Got reply from %s in %v\n", peer, duration)

			spark.Data = append(spark.Data, duration.Seconds()*1000+rand.Float64())
			if len(spark.Data) > 100 {
				spark.Data = spark.Data[1:]

			}
			max, min, avg := calculateStatistics(spark.Data)
			spark.Title = fmt.Sprintf("Max %.2f (ms),Min %.2f (ms),Avg %.2f (ms),Rt %.2f (ms)", max, min, avg, duration.Seconds()*1000+rand.Float64())
			//calculateStatistics
			if duration.Seconds()*1000+rand.Float64() > 40 && duration.Seconds()*1000+rand.Float64() < 80 {
				spark.LineColor = ui.ColorRed
				goto xx
			}
			if duration.Seconds()*1000+rand.Float64() > 10 && duration.Seconds()*1000+rand.Float64() < 40 {
				spark.LineColor = ui.ColorBlue
				goto xx
			}
			if duration.Seconds()*1000+rand.Float64() <= 10 {
				spark.LineColor = ui.ColorGreen
				goto xx
			}

		xx:
			ui.Render(plot)
		default:
			//fmt.Printf("got %+v; want echo reply", rm)
		}

		// Wait for a second before sending the next packet
		time.Sleep(300 * time.Millisecond)
	}
}
