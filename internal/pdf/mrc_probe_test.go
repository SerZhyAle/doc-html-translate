package pdf

import (
 "fmt"
 "os"
 "testing"
 "github.com/pdfcpu/pdfcpu/pkg/api"
 "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
 pdfcpulib "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
 "github.com/tannevaled/gobig2"
 "bytes"
 "context"
 "image"
 "image/color"
)

func TestMRCProbe(t *testing.T) {
 btest:=image.NewRGBA(image.Rect(0,0,2,2));for y:=0;y<2;y++ {for x:=0;x<2;x++ {btest.Set(x,y,color.RGBA{200,170,140,255})}};ftest:=image.NewRGBA(image.Rect(0,0,4,4));mtest:=image.NewGray(image.Rect(0,0,4,4));testout,_:=composeMRCPixels(btest,ftest,mtest);t.Log("synthetic output",testout.At(0,0))
 for _, path := range []string{"../../test_doc/multilingual_2026-09-29/fr-la-fontaine-fables.pdf", "../../test_doc/multilingual_2026-09-29/ja-ia-scan-weak-text-layer-1920s.pdf", "../../test_doc/pdf-1page-blackletter_Plague-Proclamation-1625.pdf", "../../test_doc/comic-textlayer-mid_Plastic-Man-v1-017.pdf"} {
  f,e:=os.Open(path); if e!=nil {t.Log(e);continue}; conf:=model.NewDefaultConfiguration();conf.Cmd=model.EXTRACTIMAGES; ctx,e:=api.ReadValidateAndOptimize(f,conf); f.Close();if e!=nil {t.Log(e);continue};
  imgs,e:=pdfcpulib.ExtractPageImages(ctx,10,true); t.Log(path,"pages",ctx.PageCount,"err",e)
  for n,i:=range imgs {t.Log(fmt.Sprintf("obj=%d %dx%d mask=%v smask=%v type=%s bits=%d name=%s filter=%s",n,i.Width,i.Height,i.HasImgMask,i.HasSMask,i.FileType,i.Bpc,i.Name,i.Filter)); if i.HasImgMask { sd,_,_:=ctx.DereferenceStreamDict(ctx.Table[n].Object); ref:=sd.Dict["Mask"]; msd,_,e:=ctx.DereferenceStreamDict(ref); t.Log("mask",e,msd.Dict); mi,e:=pdfcpulib.ExtractImage(ctx,msd,false,"mask",0,false); if mi!=nil {t.Log("mask image type",mi.FileType)};t.Log("mask err",e);dec,e:=gobig2.NewDecoderEmbedded(bytes.NewReader(msd.Raw),nil);if e==nil {m,e:=dec.Decode();if e==nil {t.Log("decoded mask",m.Bounds())}};t.Log("decode error",e) }}
  if path == "../../test_doc/multilingual_2026-09-29/fr-la-fontaine-fables.pdf" { full,e:=pdfcpulib.ExtractPageImages(ctx,10,false); if e!=nil {t.Fatal(e)}; stubs,_:=pdfcpulib.ExtractPageImages(ctx,10,true);for n,i:=range full {s:=stubs[n];i.Width=s.Width;i.Height=s.Height;i.HasImgMask=s.HasImgMask;full[n]=i}; bg,fg,ok:=mrcPair(full);if !ok {t.Fatal("not pair")}; t.Log("pair",bg.Name,bg.FileType,bg.Width,bg.Height,fg.Name,fg.FileType,fg.Width,fg.Height);mask,_:=maskForImage(ctx,fg);black:=0;for _,v:=range mask.Pix {if v<128 {black++}};t.Log("mask black",black,"total",len(mask.Pix));bgi,e:=readMRCRaster(context.Background(),bg,"../../temp");t.Log("bg pixel",bgi.At(10,10),e);fgi,e:=readMRCRaster(context.Background(),fg,"../../temp");t.Log("fg pixel",fgi.At(10,10),e);e=writeMRCComposite(context.Background(),ctx,bg,fg,"../../temp","mrc-probe.jpg");t.Log("composite",e)}
 }
}
