using System;
using System.Drawing;
using System.Drawing.Drawing2D;
using System.Drawing.Imaging;
using System.Globalization;
using System.Xml;
using System.Text.RegularExpressions;
public static class ConnectBrandRenderer {
 static float N(string value) { return float.Parse(value,CultureInfo.InvariantCulture); }
 static GraphicsPath Path(string data) {
  var tokens=Regex.Matches(data,@"[MLCZ]|-?\d+(?:\.\d+)?");
  var path=new GraphicsPath();float x=0,y=0;int i=0;
  while(i<tokens.Count){string command=tokens[i++].Value;
   if(command=="M"){x=N(tokens[i++].Value);y=N(tokens[i++].Value);path.StartFigure();}
   else if(command=="L"){float nx=N(tokens[i++].Value),ny=N(tokens[i++].Value);path.AddLine(x,y,nx,ny);x=nx;y=ny;}
   else if(command=="C"){float a=N(tokens[i++].Value),b=N(tokens[i++].Value),c=N(tokens[i++].Value),d=N(tokens[i++].Value),nx=N(tokens[i++].Value),ny=N(tokens[i++].Value);path.AddBezier(x,y,a,b,c,d,nx,ny);x=nx;y=ny;}
   else if(command=="Z")path.CloseFigure();else throw new Exception("Unsupported SVG path command");
  }return path;
 }
 public static void Render(string svg,string output,int size,bool active) {
  int factor=size<=256?8:2;int pixels=size*factor;
  using(var canvas=new Bitmap(pixels,pixels,PixelFormat.Format32bppArgb))
  using(var g=Graphics.FromImage(canvas))
  using(var silver=new LinearGradientBrush(new Point(250,140),new Point(980,1140),active?Color.FromArgb(91,235,166):Color.FromArgb(238,240,246),active?Color.FromArgb(26,173,104):Color.FromArgb(205,210,220)))
  using(var background=new LinearGradientBrush(new Point(200,140),new Point(1080,1140),Color.FromArgb(21,23,25),Color.FromArgb(9,10,11))) {
   g.SmoothingMode=SmoothingMode.AntiAlias;float extent=size<=64?1060f:1280f;g.ScaleTransform(pixels/extent,pixels/extent);g.TranslateTransform(-(1280-extent)/2,-(1280-extent)/2);
   var doc=new XmlDocument();doc.Load(svg);
   foreach(XmlNode node in doc.DocumentElement.ChildNodes){if(node.Name=="defs")continue;var e=node as XmlElement;if(e==null)continue;
    using(var shape=e.Name=="path"?Path(e.GetAttribute("d")):new GraphicsPath()) {
     if(e.Name=="circle"){float cx=N(e.GetAttribute("cx")),cy=N(e.GetAttribute("cy")),r=N(e.GetAttribute("r"));shape.AddEllipse(cx-r,cy-r,2*r,2*r);}
     string fill=e.GetAttribute("fill");if(fill!="none"&&fill!="")g.FillPath(fill.Contains("background")?background:silver,shape);
     if(e.HasAttribute("stroke")){float width=N(e.GetAttribute("stroke-width"));if(size<=32&&width<20)width=Math.Max(width,1280f/size*.7f);using(var pen=new Pen(silver,width)){pen.StartCap=LineCap.Round;pen.EndCap=LineCap.Round;pen.LineJoin=LineJoin.Round;g.DrawPath(pen,shape);}}
    }
   }
   using(var result=new Bitmap(size,size,PixelFormat.Format32bppArgb))using(var scale=Graphics.FromImage(result))using(var attrs=new ImageAttributes()){
    scale.CompositingMode=CompositingMode.SourceCopy;scale.InterpolationMode=InterpolationMode.HighQualityBicubic;scale.PixelOffsetMode=PixelOffsetMode.HighQuality;attrs.SetWrapMode(WrapMode.TileFlipXY);
    scale.DrawImage(canvas,new Rectangle(0,0,size,size),0,0,pixels,pixels,GraphicsUnit.Pixel,attrs);result.Save(output,ImageFormat.Png);
   }
  }
 }
}
