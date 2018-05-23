package nextcloud

import(
	"encoding/xml"
	"log"
	"fmt"
)


/*
 * Utility to manage webdav responses. The Response struct should always be instantiated
 * through the `NewResponseFromXml`
 */

type Response struct{
	XMLName xml.Name `xml:"multistatus"`
	Files []*File     `xml:"response"`
}

type File struct{
	Href string `xml:"href"`
	LastModified string `xml:"propstat>prop>getlastmodified"`
	FileId int `xml:"propstat>prop>fileid"`
	ContentType string `xml:"propstat>prop>getcontenttype"`
}

func (f *File) IsDir() bool {
	return f.ContentType == ""
}

func (r *Response) RemoveFirstFile() {
	if len(r.Files) > 1 {
		r.Files = r.Files[1:len(r.Files)]
	} else {
		r.Files = r.Files[1:]
	}
}
func (f *File) String() string{
	return fmt.Sprintf("File:\nHref: %s, LastMod: %s, FileId: %d, ContentType: %s", f.Href, f.LastModified,
		f.FileId, f.ContentType)
}

func NewResponseFromXML(xmlData []byte) (*Response, error) {
	resp := &Response{}
	err := xml.Unmarshal(xmlData, resp)

	if err != nil{
		log.Fatal("Unable to unmarshal the XML received ", err)
		return nil, err
	}

	return resp, nil
}
