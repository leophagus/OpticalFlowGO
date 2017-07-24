// optical flow using Lucas Kanade, dataflow conversion
// 2016-08-14 20:27:51 leophagus
//
//
package main

import "fmt"
import "os"
import "strconv"
import "io"
import "bufio"
import "math"
import "time"
import "runtime"

const WIDTH = 1280
const HEIGHT = 800
const WIN = 25

var maxRad float64 = 0.0

func readPpm (fnme string, bufChan chan byte, quit chan string) {
  t0 := time.Now ()
  fmt.Println("readPpm", fnme)
  ifile, err := os.Open (fnme)
  if err != nil {
    //fmt.Println ("Failed to open file", fnme)
    quit <- "readPpm Failed to open file"
    return
  }
  defer ifile. Close ()

  irdr := bufio.NewReader (ifile)
  //P6\nWidth Height\n255\nbuf

  var p, w, h, d string
  p, err = irdr. ReadString ('\n')
  if p != "P6\n" || err != nil {
    quit <- "readPpm Failed to get P6"
    return
  }
  w, err = irdr. ReadString (' ')
  h, err = irdr. ReadString ('\n')
  d, err = irdr. ReadString ('\n')
  fmt.Println ("Got (strings)", w, h, d)

  w = w[:len(w)-1]
  h = h[:len(h)-1]
  d = d[:len(d)-1]

  var width, height int
  width, err = strconv.Atoi (w)
  height, err = strconv.Atoi (h)
  fmt. Println ("Got width", width, " height", height, " depth ", d)
  t1 := time.Now ()
  fmt.Printf ("readPpm in %v\n", t1.Sub(t0))

  if width != WIDTH {
    quit <- "Width mismatch"
    return
  }
  if height != HEIGHT {
    quit <- "Height mismatch"
    return
  }

  var n int
  buf := make([]byte, width * height * 3)
  // Read seems to stop at some byte. Read only 4081 with foo_p.ppm
  //n, err = irdr. Read (buf)
  n, err = io. ReadFull (irdr, buf)
  if n==0 || err != nil {
    quit <- "readPpm Failed to read buf"
    return
  }
  fmt.Println ("Read", n, "bytes")
  //for _, b := range buf {
  //  bufChan <- b
  //}
  for i:=0; i<n; i+=3 {
    bufChan <- buf [i]
  }

  close (bufChan)

}

func writePpm (fnme string,
               bufChan chan byte, done chan bool, quit chan string) {
  t0 := time.Now ()
  file, err := os.OpenFile (fnme, os.O_WRONLY|os.O_CREATE, 0666)
  if err != nil {
    quit <- "writePpm failed to write file: " + fnme
    return 
  }
  defer file. Close ()

  writer := bufio.NewWriter (file)
  writer.WriteString("P6\n")
  writer.WriteString(strconv.Itoa(WIDTH))
  writer.WriteString(" ")
  writer.WriteString(strconv.Itoa(HEIGHT))
  writer.WriteString("\n255\n")

  var n int
  for b := range bufChan {
    writer.WriteByte (b)
    n ++
  }
  writer.Flush ()

  fmt.Println ("Wrote ", n, "bytes to", fnme)

  t1 := time.Now ()
  fmt.Printf ("writePpm in %v\n", t1.Sub(t0))
  done <- true
} 

func lineBuffer (f0 chan byte, f0Col chan [WIN] byte) {
  var lb1 [WIN][WIDTH]byte
  var col [WIN]byte
  for r := 0; r < HEIGHT; r++ {
    for c := 0; c < WIDTH; c++ {

      for i := 0; i < WIN-1; i++ {
        lb1 [i][c] = lb1 [i+1][c]
        col [i] = lb1 [i][c]
      }
      pix := <- f0
      col [WIN-1] = pix
      lb1 [WIN-1][c] = pix

      //fmt.Println ("lb",r,c,pix,col)
      f0Col <- col
    }
  }

}

func computeSums (f0Col chan [WIN]byte, f1Col chan [WIN]byte,
                  ixixChan chan int,
                  ixiyChan chan int,
                  iyiyChan chan int,
                  dixChan chan int,
                  diyChan chan int) {

  var f0Win, f1Win [WIN*WIN]int
  var f0Col_, f1Col_ [WIN]byte
  var ixix, iyiy, ixiy, dix, diy int = 0,0,0,0,0

  {
    for i:= range f0Win {
      f0Win [i] = 0
      f1Win [i] = 0
    }
    for i:= range f0Col_ {
      f0Col_[i] = 0
      f1Col_[i] = 0
    }
  }

  for r := 0; r < HEIGHT; r++ {
    for c := 0; c < WIDTH; c++ {
      //fmt.Println("computeSums",r,c)

      f1Col_ = <- f1Col
      f0Col_ = <- f0Col

      for wr := 1; wr < WIN-1; wr++ {

        var wcl int = 1
        var cIx_left, cIy_left, del_left int = 0,0,0

        if r==0 && c<WIN-1 {
          cIx_left=0; cIy_left=0; del_left=0;
        } else {
          cIx_left = (f0Win [wr*WIN + wcl+1] - f0Win [wr*WIN + wcl-1])/2;
          cIy_left = (f0Win [ (wr+1)*WIN +wcl] - f0Win [ (wr-1)*WIN + wcl])/2;
          del_left = (f0Win [wr*WIN + wcl] - f1Win [wr*WIN + wcl]);
        }

        var cIx_right int = ( int(f0Col_ [wr]) - f0Win [wr*WIN + WIN-2]) / 2 ;
        var cIy_right int = (f0Win [ (wr+1)*WIN + WIN-1] - f0Win [ (wr-1)*WIN + WIN-1])/2;
        var del_right int = (f0Win [wr*WIN + WIN-1] - f1Win [wr*WIN + WIN-1]);

        ixix += (cIx_right * cIx_right - cIx_left * cIx_left);
        iyiy += (cIy_right * cIy_right - cIy_left * cIy_left);
        ixiy += (cIx_right * cIy_right - cIx_left * cIy_left);
        dix  += (del_right * cIx_right - del_left * cIx_left);
        diy  += (del_right * cIy_right - del_left * cIy_left);
        //fmt.Println("wr", wr, cIx_left, cIy_left, del_left, cIx_right, cIy_right, del_right, ixix, ixiy, iyiy, dix, diy)
      }

      ixixChan <- ixix
      ixiyChan <- ixiy
      iyiyChan <- iyiy
      dixChan <- dix
      diyChan <- diy
      //fmt.Println("f0Col_", f0Col_, "f1Col_", f1Col_);
      //fmt.Println("cs",r,c,"f0Col_", f0Col_, "f1Col_", f1Col_, ixix, ixiy, iyiy, dix, diy)

      var i, j int
      for i = 0; i < WIN; i++ {
        for j = 0; j < WIN - 1; j++ {
          f0Win [i * WIN + j] = f0Win [i * WIN + j + 1];
          f1Win [i * WIN + j] = f1Win [i * WIN + j + 1];
        }
      }

      for i=0; i < WIN; i++ {
        f0Win  [i*WIN + WIN-1] = int (f0Col_ [i]);
        f1Win  [i*WIN + WIN-1] = int (f1Col_ [i]);
      }

    }
  }
}

func computeFlow (ixixChan chan int,
                  ixiyChan chan int,
                  iyiyChan chan int,
                  dixChan chan int,
                  diyChan chan int,
                  fxChan chan float32,
                  fyChan chan float32) {
  for r := 0; r < HEIGHT; r++ {
    for c := 0; c < WIDTH; c++ {
      ixix := <- ixixChan
      ixiy := <- ixiyChan
      iyiy := <- iyiyChan
      dix := <- dixChan
      diy := <- diyChan
      var fx, fy, det, i00, i01, i10, i11 float32

      // matrix inv
      det = float32(ixix * iyiy - ixiy * ixiy)
      if det <= 1.0 {
        fx = 0.0
        fy = 0.0
      } else {
        i00 = float32(iyiy) / det
        i01 = float32(-ixiy) / det
        i10 = float32(-ixiy) / det
        i11 = float32(ixix) / det

        fx = i00 * float32(dix) + i01 * float32(diy)
        fy = i10 * float32(dix) + i11 * float32(diy)
      }
      fxChan <- fx
      fyChan <- fy
      //fmt.Println ("sums",ixix,ixiy,iyiy,dix,diy,i00, i01, i10, i11);
      //fmt.Println("fl",r,c,fx,fy)
    }
  }
}

func getPseudoColorInt (pix byte, fx float32, fy float32) (r, g, b byte) {
  // normalization factor is key for good visualization. Make this aut-ranging
  // or controllabel from the host TODO
  
  var normFac int = 128/2;

  var y int = 127 + int(fy) * normFac
  var x int = 127 + int(fx) * normFac
  if (y>255) { y=255 }
  if (y<0) { y=0 }
  if (x>255) { x=255 }
  if (x<0) { x=0 }

  var r_, g_, b_ byte
  if (x > 127) {
    if (y < 128) {
      // 1 quad
      r_ = byte(x - 127 + (127-y)/2)
      g_ = byte((127 - y)/2)
      b_ = 0
    } else {
      // 4 quad
      r_ = byte(x - 127)
      g_ = 0
      b_ = byte(y - 127)
    }
  } else {
    if (y < 128) {
      // 2 quad
      r_ = byte((127 - y)/2)
      g_ = byte(127 - x + (127-y)/2)
      b_ = 0
    } else {
      // 3 quad
      r_ = 0
      g_ = byte(128 - x)
      b_ = byte(y - 127)
    }
  }

  r = pix/2 + r_/2 
  g = pix/2 + g_/2 
  b = pix/2 + b_/2
  return
}

func getColor (pix byte, fx float32, fy float32) (r, g, b byte) {
  rad := math.Sqrt ( float64(fx*fx + fy*fy))
  if (rad > maxRad) { maxRad = rad}
  if rad > 1.0 { rad = 1.0 }
  out_p := byte (rad * 255.0)
  r = out_p
  g = out_p
  b = out_p
  return
}
 
func getOutPix (fxChan chan float32,
                fyChan chan float32,
                //f0PixChan chan byte,
                outPix chan byte) {

  for r := 0; r < HEIGHT; r++ {
    for c := 0; c < WIDTH; c++ {

      fx :=  <- fxChan
      fy :=  <- fyChan
      //pix := <- f0PixChan
      o_r,o_g,o_b := getPseudoColorInt (0, fx, fy)
      //o_r,o_g,o_b := getColor (0, fx, fy)
      outPix <- o_r
      outPix <- o_g
      outPix <- o_b
      //fmt.Println("getOUtPix",r,c)
    }
  }
  fmt.Println("maxRad", maxRad)
  close (outPix)
}

func main () {

  runtime.GOMAXPROCS(1)

  if (len (os.Args) != 4) {
    fmt.Println ("Usage: optflow frame0.ppm frame1.ppm frameOut.ppm")
    return
  }
  inFnme0 := os.Args [1]
  inFnme1 := os.Args [2]
  outFnme := os.Args [3]
  //fmt.Println ("Got", inFnme, outFnme)

  quit := make (chan string)
  done := make (chan bool)

  f0 := make (chan byte)
  f1 := make (chan byte)
  fOut := make (chan byte)
  f0Col := make (chan [WIN]byte)
  f1Col := make (chan [WIN]byte)
  ixix := make (chan int)
  ixiy := make (chan int)
  iyiy := make (chan int)
  dix := make (chan int)
  diy := make (chan int)
  fx := make (chan float32)
  fy := make (chan float32)

  t0 := time.Now ()

  go readPpm (inFnme0,  f0, quit)
  go readPpm (inFnme1,  f1, quit)

  go lineBuffer (f0, f0Col)
  go lineBuffer (f1, f1Col)

  go computeSums (f0Col, f1Col, ixix, ixiy, iyiy, dix, diy)
  go computeFlow (ixix, ixiy, iyiy, dix, diy, fx, fy)
  go getOutPix (fx, fy, fOut)

  go writePpm (outFnme, fOut, done, quit)

  select {
  case <-done:
    fmt.Println ("Got done")
  case s:= <-quit:
    fmt.Println ("ERROR:", s)
  }

  t1 := time.Now ()
  fmt.Printf ("Optflow computed in %v\n", t1.Sub(t0))
}
