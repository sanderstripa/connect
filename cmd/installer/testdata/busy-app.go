// This inert process only holds its executable open for the installer lock test.
package main
import "time"
func main(){time.Sleep(10*time.Minute)}
