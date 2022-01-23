package images

import "os"

var(
	ImagesStoreDir = "/var/lib/mydocker/images"
)

func init(){
	 _= os.MkdirAll(ImagesStoreDir, 0755)
}
