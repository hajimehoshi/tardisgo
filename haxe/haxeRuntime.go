// Copyright 2014 Elliott Stoneham and The TARDIS Go Authors
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package haxe

// Runtime Haxe code for Go, which may eventually become a haxe library when the system settles down.
// TODO All runtime class names are currently carried through if the haxe code uses "import tardis.Go;" and some are too generic,
// others, like Int64, will overload the Haxe standard library version for some platforms, which may cause other problems.
// So the haxe Class names eventually need to be prefaced with "Go" to ensure there are no name-clashes.
// TODO using a library would reduce some of the compilation overhead of always re-compiling this code.
// However, there are references to Go->Haxe generated classes, like "Go", that would need to be managed somehow.
// TODO consider merging and possibly renaming the Deep and Force classes as they both hold general utility code

var haxeruntime = `

// TODO: consider putting these go-compatibiliy classes into a separate library for general Haxe use when calling Go

class Force { // TODO maybe this should not be a separate haxe class, as no non-Go code needs access to it

	public static inline function toUint8(v:Int):Int {
		return v & 0xFF;
	}	
	public static inline function toUint16(v:Int):Int {
		return v & 0xFFFF;
	}	
	public static inline function toUint32(v:Int):Int { 
		#if js
			return v >>> untyped __js__("0"); // using GopherJS method (with workround to stop it being optimized away by Haxe)
		#elseif php
       		return v & untyped __php__("0xffffffff");
		#else
			return v; 
		#end
	}	
	public static inline function toUint64(v:GOint64):GOint64 {
		return v;
	}	
	public static inline function toInt8(v:Int):Int {
		var r:Int = v & 0xFF;
		if ((r & 0x80) != 0){ // it should be -ve
			return -1 - 0xFF + r;
		}
		return r;
	}	
	public static inline function toInt16(v:Int):Int {
		var r:Int = v & 0xFFFF;
		if ((r & 0x8000) != 0){ // it should be -ve
			return -1 - 0xFFFF + r;
		}
		return r;
	}	
	public static inline function toInt32(v:Int):Int {
		#if js 
			return v >> untyped __js__("0"); // using GopherJS method (with workround to stop it being optimized away by Haxe)
		#elseif php
			//see: http://stackoverflow.com/questions/300840/force-php-integer-overflow
     		v = (v & untyped __php__("0xFFFFFFFF"));
 		    if( (v & untyped __php__("0x80000000")) != 0)
		        v = -((~v & untyped __php__("0xFFFFFFFF")) + 1);
		    return v;
		#else
			return v;
		#end
	}	
	public static inline function toInt64(v:GOint64):GOint64 { // this in case special handling is required for some platforms
		return v;
	}	
	public static inline function toInt(v:Dynamic):Int { // get an Int from a Dynamic variable (uintptr is stored as Dynamic)
		if (!Reflect.isObject(v))  			// simple type, so leave quickly and take defaults 
			return v; 
		else
			if(Std.is(v,Interface)) {
				v=v.val; // it is in an interface, so get the value
				if (!Reflect.isObject(v))  			// simple type from inside an interface, so take defaults 
					return v; 
				else								// it should be an Int64 from inside an Interface
					return GOint64.toInt(v);	
			} else								// it should be an Int64 if not an interface
				return GOint64.toInt(v);	
	}
	public static inline function toFloat(v:Float):Float {
		// neko target platform requires special handling because it auto-converts whole-number Float into Int without asking
		// see: https://github.com/HaxeFoundation/haxe/issues/1282 which was marked as closed, but was not fixed as at 2013.9.6
		#if neko
			if(Std.is(v,Int))
				return v + 2.2251e-308; // add the smallest value possible for a 64-bit float to ensure neko doesn't still think it is an int
			else
				return v;
		#else
			return v;
		#end
	}	
	public static function uintCompare(x:Int,y:Int):Int { // +ve if uint(x)>unint(y), 0 equal, else -ve 
			if(x==y) return 0; // simple case first for speed TODO is it faster with this in or out?
			if(x>=0) {
				if(y>=0){ // both +ve so normal comparison
					return (x-y);
				}else{ // y -ve and so larger than x
					return -1;
				}
			}else { // x -ve
				if(y>=0){ // -ve x larger than +ve y
					return 1;
				}else{ // both are -ve so the normal comparison
					return (x-y); //eg -1::-7 gives -1--7 = +6 meaning -1 > -7
				}
			}
	}
	private static function checkIntDiv(x:Int,y:Int,byts:Int):Int { // implement the special processing required by Go
		var r:Int=y;
		switch(y) {
		case 0:
			Scheduler.panicFromHaxe("attempt to divide integer value by 0"); 
		case -1:
			switch (byts) {
			case 1:
				if(x== -128) r=1; // special case in the Go spec
			case 2:
				if(x== -32768) r=1; // special case in the Go spec
 			case 4:
				if(x== -2147483648) r=1; // special case in the Go spec
			default:
				// noOp - 0 => unsigned
			}
		}
		return r;
	}
	//TODO maybe optimize by not passing the special value and having multiple versions of functions
	public static function intDiv(x:Int,y:Int,sv:Int):Int {
		y = checkIntDiv(x,y,sv);
		if(y==1) return x; // x div 1 is x
		if((sv>0)||((x>0)&&(y>0))){ // signed division will work (even though it may be unsigned)
			var f:Float=  cast(x,Float) / cast(y,Float);
			return f>=0?Math.floor(f):Math.ceil(f);
		} else { // unsigned division 
			return GOint64.toInt(GOint64.div(GOint64.make(0x0,x),GOint64.make(0x0,y),false));
		}
	}
	public static function intMod(x:Int,y:Int,sv:Int):Int {
		y = checkIntDiv(x,y,sv);
		if(y==1) return 0; // x mod 1 is 0
		if((sv>0)||((x>0)&&(y>0))){ // signed mod will work (even though it may be unsigned)
			return x % y;
		} else { // unsigned mod (do it in 64 bits to ensure unsigned)
			return GOint64.toInt(GOint64.mod(GOint64.make(0x0,x),GOint64.make(0x0,y),false));
		}
	}
	public static function floatDiv(x:Float,y:Float):Float {
		if(y==0.0)
			Scheduler.panicFromHaxe("attempt to divide float value by 0"); 
		return x/y;
	}
	public static function floatMod(x:Float,y:Float):Float {
		if(y==0.0)
			Scheduler.panicFromHaxe("attempt to modulo float value by 0"); 
		return x%y;
	}

	public static function toUTF8length(gr:Int,s:String):Int {
		return "字".length==3 ? s.length : toUTF8slice(gr,s).len(); // no need to unpack the string if already UTF8
	}
	// return the UTF8 version of a potentiallly UTF16 string in a Slice
	public static function toUTF8slice(gr:Int,s:String):Slice {
		var a:Array<Int> = new Array<Int>();
		for(i in 0...s.length){
				var t:Null<Int>=s.charCodeAt(i) ;
				if(t==null) 
					Scheduler.panicFromHaxe("Haxe runtime Force.toUTF8slice() unexpected null encountered");
				else
					a[i]=Std.int(t) ;
		}
		if ( "字".length==3 ) { // already UTF8 encoded
			var sl:Slice = new Slice(new Pointer(new Object(a.length)),0,-1,a.length,1);
			for(i in 0...a.length)
				sl.itemAddr(i).store_uint8(a[i]);
			return sl;
		}else{
			var sl:Slice = new Slice(new Pointer(new Object(a.length<<1)),0,-1,a.length,2);
			for(i in 0...a.length)
				sl.itemAddr(i).store_uint16(a[i]);
			var v1:Slice=Go_haxegoruntime_UTF16toRunes.callFromRT(gr,sl);
			return Go_haxegoruntime_RunesToUTF8.callFromRT(gr,v1);
		}
	}
	public static function toRawString(gr:Int,sl:Slice):String {
		var ret:String="";
		if ( "字".length==1 ) { // needs to be translated to UTF16
			var v1:Slice=Go_haxegoruntime_UTF8toRunes.callFromRT(gr,sl);
			sl=Go_haxegoruntime_RunesToUTF16.callFromRT(gr,v1);
			for(i in 0...sl.len()) {
				ret += String.fromCharCode( sl.itemAddr(i).load_uint16() );
			}
			return ret;
		}
		for(i in 0...sl.len()) {
			ret += String.fromCharCode( sl.itemAddr(i).load_uint8() );
		}
		return ret;
	}
	
}

// Object code
// a single type of Go object
@:keep
class Object { // this implementation will improve with typed array access
	// Simple! 1 address per byte, non-Int types are always on 4-byte
	
	#if (js && dataview)
		private var dVec4:haxe.ds.Vector<Dynamic>; // on 4-byte boundaries 
		private var arrayBuffer:js.html.ArrayBuffer;
		private var dView:js.html.DataView;
	#else
		private var dVec4:haxe.ds.Vector<Dynamic>; 
		private var iVec:haxe.ds.Vector<Int>; 
	#end
	private var length:Int;

	public inline function new(byteSize:Int){ // size is in bytes
		#if (js && dataview)
			dVec4 = new haxe.ds.Vector<Dynamic>(1+(byteSize>>2)); 
			arrayBuffer = new js.html.ArrayBuffer(byteSize);
			if(byteSize>0)
				dView = new js.html.DataView(arrayBuffer,0,byteSize); // complains if size is 0, TODO review
		#else
			dVec4 = new haxe.ds.Vector<Dynamic>(1+(byteSize>>2)); 
			iVec = new haxe.ds.Vector<Int>(byteSize);
		#end
		length = byteSize;
	}
	public function isEqual(off:Int,target:Object,tgtOff:Int):Bool { // TODO check if correct, used by interface{} value comparison
		trace("isEqual");
		if((this.length-off)!=(target.length-tgtOff)) return false;
		for(i in 0...(this.length-off)) {
			if(this.get(i+off)!=target.get(i+tgtOff))
				return false;
			if(this.get_uint8(i+off)!=target.get_uint8(i+tgtOff))
				return false;
		}
		return true;
	}
	private static function objBlit(src:Object,srcPos:Int,dest:Object,destPos:Int,size:Int):Void{
		#if (js && dataview)
			if((size&3==0)&&(srcPos&3==0)&&(destPos&3==0)) {
				var i:Int=0;
				var s:Int=srcPos;
				var d:Int=destPos;
				while(i<size){
					dest.set_uint32(d,src.get_uint32(s)); 
					dest.set(d,src.get(s));
					i+=4;
					s+=4;
					d+=4;
				}
			}
			else{
				var s:Int=srcPos;
				var d:Int=destPos;
				for(i in 0...size) {
					dest.set_uint8(d,src.get_uint8(s));
					if((s&3)==0){ 
						dest.set(d,src.get(s));
					}
					s+=1;
					d+=1;
				}
			}
		#else
			haxe.ds.Vector.blit(src.dVec4,srcPos>>2, dest.dVec4, destPos>>2, 1+(size>>2)); 
			haxe.ds.Vector.blit(src.iVec,srcPos, dest.iVec, destPos, size); 
		#end
	}
	public inline function get_object(size:Int,from:Int):Object { // TODO SubObj class that is effectively a pointer?
		var so:Object = new Object(size);
		objBlit(this,from, so, 0, size); 
		return so;
	}
	public function set_object(size:Int, to:Int, from:Object):Void {
		#if php
			if(!Std.is(from,Object)) { 
				//Scheduler.panicFromHaxe("Object.set_object() from parameter is not an Object - Value: "+Std.string(from)+" Type: "+Type.typeof(from));
				return; // treat as null object (seen examples have been integer 0)
			}
		#end
		objBlit(from,0,this,to,size);
	}
	public inline function copy():Object{
		return this.get_object(length,0);
	}
	public inline function get(i:Int):Dynamic {
			return dVec4[i>>2];
	}
	public inline function get_bool(i:Int):Bool { 
		#if (js && dataview)
			return dView.getUint8(i)==0?false:true;
		#else
			var r:Int=iVec[i]; 
			#if (js || php || neko ) 
				return r==null?false:(r==0?false:true); 
			#else 
				return r==0?false:true; 
			#end
		#end
	}
	public inline function get_int8(i:Int):Int { 
		#if (js && dataview)
			return dView.getInt8(i);
		#else
			var r:Int=iVec[i]; 
			#if (js || php || neko ) return r==null?0:0|r; #else return r; #end
		#end
	}
	public inline function get_int16(i:Int):Int { 
		#if (js && dataview)
			return dView.getInt16(i);
		#else
			var r:Int=iVec[i]; 
			#if (js || php || neko ) return r==null?0:0|r; #else return r; #end
		#end
	}
	public inline function get_int32(i:Int):Int {
		#if (js && dataview)
			return dView.getInt32(i);
		#else
			var r:Int=iVec[i]; 
			#if (js || php || neko ) return r==null?0:0|r; #else return r; #end
		#end
	}
	public inline function get_int64(i:Int):GOint64 {
		// TODO optimize for dataview
		var r:GOint64=get(i); 
		return r==null?GOint64.ofInt(0):r;			
	} 
	public inline function get_uint8(i:Int):Int { 
		#if (js && dataview)
			return dView.getUint8(i);
		#else
			var r:Int=iVec[i]; 
			#if (js || php || neko ) return r==null?0:0|r; #else return r; #end
		#end
	}
	public inline function get_uint16(i:Int):Int {
		#if (js && dataview)
			return dView.getUint16(i);
		#else
			var r:Int=iVec[i]; 
			#if (js || php || neko ) return r==null?0:0|r; #else return r; #end
		#end
	}
	public inline function get_uint32(i:Int):Int {
		#if (js && dataview)
			return dView.getUint32(i);
		#else
			var r:Int=iVec[i]; 
			#if (js || php || neko ) return r==null?0:0|r; #else return r; #end
		#end
	}
	public inline function get_uint64(i:Int):GOint64 { 
		// TODO optimize for dataview
		var r:GOint64=get(i); 
		return r==null?GOint64.ofInt(0):r;			
	} 
	public inline function get_uintptr(i:Int):Dynamic { // uintptr holds Haxe objects
		return get(i); 
	} 
	public inline function get_float32(i:Int):Float { 
		#if (js && dataview)
			return dView.getFloat32(i);
		#else
			var r:Float=get(i); 
			#if (js || php || neko ) 
				return r==null?0.0:r; 
			#else 
				return r; 
			#end
		#end
	}
	public inline function get_float64(i:Int):Float { 
		#if (js && dataview)
			return dView.getFloat64(i);
		#else
			var r:Float=get(i); 
			#if (js || php || neko ) 
				return r==null?0.0:r; 
			#else 
				return r;
			#end 
		#end
	}
	public inline function get_complex64(i:Int):Complex {
		// TODO optimize for dataview
		var r:Complex=get(i); 
		return r==null?new Complex(0.0,0.0):r;			
	}
	public inline function get_complex128(i:Int):Complex { 
		// TODO optimize for dataview
		var r:Complex=get(i); 
		return r==null?new Complex(0.0,0.0):r;			
	}
	public inline function get_string(i:Int):String { 
		var r:String=get(i); 
		#if (js || php || neko ) return r==null?"":r; #else return r; #end
	}
	public inline function set(i:Int,v:Dynamic):Void { 
		dVec4[i>>2]=v;
	}
	public inline function set_bool(i:Int,v:Bool):Void { 
		#if (js && dataview)
			dView.setUint8(i,v?1:0);
		#else
			iVec[i]=v?1:0; 
		#end
	} 
	public inline function set_int8(i:Int,v:Int):Void { 
		#if (js && dataview)
			dView.setInt8(i,v);
		#else
			iVec[i]=v; 
		#end
	}
	public inline function set_int16(i:Int,v:Int):Void { 
		#if (js && dataview)
			dView.setInt16(i,v);
		#else
			iVec[i]=v; 
		#end
	}
	public inline function set_int32(i:Int,v:Int):Void { 
		#if (js && dataview)
			dView.setInt32(i,v);
		#else
			iVec[i]=v; 
		#end
	}
	public inline function set_int64(i:Int,v:GOint64):Void { 
		set(i,v); 
	} 
	public inline function set_uint8(i:Int,v:Int):Void { 
		#if (js && dataview)
			dView.setUint8(i,v);
		#else
			iVec[i]=v; 
		#end
	}
	public inline function set_uint16(i:Int,v:Int):Void { 
		#if (js && dataview)
			dView.setUint16(i,v);
		#else
			iVec[i]=v; 
		#end
	}
	public inline function set_uint32(i:Int,v:Int):Void { 
		#if (js && dataview)
			dView.setUint32(i,v);
		#else
			iVec[i]=v; 
		#end
	}
	public inline function set_uint64(i:Int,v:GOint64):Void { 
		set(i,v); 
	} 
	public inline function set_uintptr(i:Int,v:Dynamic):Void { 
		set(i,v); 
	}
	public inline function set_float32(i:Int,v:Float):Void {
		#if (js && dataview)
			dView.setFloat32(i,v);
		#else
			set(i,v); 
		#end	
	}
	public inline function set_float64(i:Int,v:Float):Void {
	 	#if (js && dataview)
			dView.setFloat64(i,v);
		#else
			set(i,v); 
		#end	
	}
	
	public inline function set_complex64(i:Int,v:Complex):Void { 
		set(i,v); 
	} 
	public inline function set_complex128(i:Int,v:Complex):Void { 
		set(i,v); 
	} 
	public inline function set_string(i:Int,v:String):Void { 
		set(i,v); 
	}
	private inline static function str(v:Dynamic):String{
		return v==null?"":Std.string(v);
	}
	public inline function toString(addr:Int=0,count:Int=-1):String{
		if(count==-1) count=this.length;
		var ret:String =  "{";
		for(i in 0...count){
			if(i>0) ret = ret + ",";
			if((addr)&3==0) ret += str(get(addr));
			ret = ret+"<"+Std.string(get_uint8(addr))+">";
			addr = addr+1;
		}
		return ret+"}";
	}
}
@:keep
class Pointer { 
	private var obj:Object; // reference to the object holding the value
	private var off:Int; // the offset into the object, if any 

	public inline function new(from:Object){
		obj = from; 
		off = 0;
	}
	public inline function addr(byteOffset:Int):Pointer {
		var ret:Pointer = new Pointer(this.obj);
		ret.off = this.off+byteOffset;
		return ret;
	}
	public inline function fieldAddr(byteOffset:Int):Pointer {
		return this.addr(byteOffset);
	}
	public inline function copy():Pointer {
		return this;
	}
	public inline function isEqual(other:Pointer):Bool{
		return obj.isEqual(this.off,other.obj,other.off);
	}
	public inline function load_object(sz:Int):Object { 
		return obj.get_object(sz,off);
	}
	public inline function load():Dynamic {
		return obj.get(off);
	}
	public inline function load_bool():Bool { 
		return obj.get_bool(off);
	}
	public inline function load_int8():Int { 
		return obj.get_int8(off);
	}
	public inline function load_int16():Int { 
		return obj.get_int16(off);
	}
	public inline function load_int32():Int {
		return obj.get_int32(off);
	}
	public inline function load_int64():GOint64 { 
		return obj.get_int64(off);
	} 
	public inline function load_uint8():Int { 
		return obj.get_uint8(off);
	}
	public inline function load_uint16():Int {
		return obj.get_uint16(off);
	}
	public inline function load_uint32():Int {
		return obj.get_uint32(off);
	}
	public inline function load_uint64():GOint64 { 
		return obj.get_uint64(off);
	} 
	public inline function load_uintptr():Dynamic { 
		return obj.get_uintptr(off);
	} 
	public inline function load_float32():Float { 
		return obj.get_float32(off);
	}
	public inline function load_float64():Float { 
		return obj.get_float64(off);
	}
	public inline function load_complex64():Complex {
		return obj.get_complex64(off);
	}
	public inline function load_complex128():Complex { 
		return obj.get_complex128(off);
	}
	public inline function load_string():String { 
		return obj.get_string(off);
	}
	public function store_object(sz:Int,v:Object):Void {
		obj.set_object(sz,off,v);
	}
	public function store(v:Dynamic):Void {
		obj.set(off,v);
	}
	public inline function store_bool(v:Bool):Void { obj.set_bool(off,v); }
	public inline function store_int8(v:Int):Void { obj.set_int8(off,v); }
	public inline function store_int16(v:Int):Void { obj.set_int16(off,v); }
	public inline function store_int32(v:Int):Void { obj.set_int32(off,v); }
	public inline function store_int64(v:GOint64):Void { obj.set_int64(off,v); }  
	public inline function store_uint8(v:Int):Void { obj.set_uint8(off,v); }
	public inline function store_uint16(v:Int):Void { obj.set_uint16(off,v); }
	public inline function store_uint32(v:Int):Void { obj.set_uint32(off,v); }
	public inline function store_uint64(v:GOint64):Void { obj.set_uint64(off,v); } 
	public inline function store_uintptr(v:Dynamic):Void { obj.set_uintptr(off,v); }
	public inline function store_float32(v:Float):Void { obj.set_float32(off,v); }
	public inline function store_float64(v:Float):Void { obj.set_float64(off,v); }
	public inline function store_complex64(v:Complex):Void { obj.set_complex64(off,v); }
	public inline function store_complex128(v:Complex):Void { obj.set_complex128(off,v); }
	public inline function store_string(v:String):Void { obj.set_string(off,v); }
	public inline function toString(sz:Int):String {
		return obj.toString(off,sz);
	}
}

// Unsafe Pointer code

@:keep
class UnsafePointer  {  // Unsafe Pointers are not yet supported, but Go library code requires that they can be created
	public function new(x:Dynamic){
	}
}

@:keep
class Slice {
	private var baseArray:Pointer;
	public var itemSize:Int; // for the size of each item in bytes 
	private var start:Int;
	private var end:Int;
	private var capacity:Int;
	public var length:Int; // could make this a function access, but it never changes and is used a lot
	
	public function new(fromArray:Pointer, low:Int, high:Int, ularraysz:Int, isz:Int) {
		baseArray = fromArray;
		itemSize = isz;
		if(baseArray==null) {
			start = 0;
			end = 0;
			capacity = 0;
		} else {
			if( low<0 ) Scheduler.panicFromHaxe( "new Slice() low bound -ve"); 
			capacity = ularraysz - low; // the capacity of what remains of the array
			if(high==-1) high = ularraysz; //default upper bound is the capacity of the underlying array
			if( high > ularraysz ) Scheduler.panicFromHaxe("new Slice() high bound exceeds underlying array length"); 
			if( low>high ) Scheduler.panicFromHaxe("new Slice() low bound exceeds high bound"); 
			start = low;
			end = high;
		}
		length = end-start;
	} 
	public function subSlice(low:Int, high:Int):Slice {
		if(high==-1) high = length; //default upper bound is the length of the current slice
		return new Slice(baseArray,low+start,high+start,capacity+low+start,itemSize);
	}
	public function append(newEnt:Slice):Slice{
		// ignore capacity filling optimization for now TODO
		var newObj:Object = new Object((length+newEnt.len())*itemSize);
		for(i in 0...length) 
			newObj.set_object(itemSize,i*itemSize,this.itemAddr(i).load_object(itemSize));
		for(i in 0...newEnt.len())
			newObj.set_object(itemSize,length*itemSize+i*itemSize,newEnt.itemAddr(i).load_object(itemSize));
		return new Slice(new Pointer(newObj),0,length+newEnt.len(),length+newEnt.len(),itemSize);
	}
	public function copy(source:Slice):Int{
		var copySize:Int=this.len();
		if(source.len()<this.len()) 
			copySize=source.len(); 
		if(this.baseArray==source.baseArray){ // copy within the same slice
			if(this.start<=source.start){
				for(i in 0...copySize)
					this.itemAddr(i).store_object(itemSize,source.itemAddr(i).load_object(itemSize));
			}else{
				for(i in copySize...0)
					this.itemAddr(i).store_object(itemSize,source.itemAddr(i).load_object(itemSize));
			}
		}else{
			for(i in 0...copySize)
				this.itemAddr(i).store_object(itemSize,source.itemAddr(i).load_object(itemSize));
		}
		return copySize;
	}
	//public inline function getAt(idx:Int):Dynamic {
	//	//if (idx<0 || idx>=(end-start)) Scheduler.panicFromHaxe("Slice index out of range for getAt()");
	//	return baseArray.addr(idx+start).load();
	//}
	//public inline function setAt(idx:Int,v:Dynamic) {
	//	//if (idx<0 || idx>=(end-start)) Scheduler.panicFromHaxe("Slice index out of range for setAt()");
	//	baseArray.addr(idx+start).store(v); // this code relies on the object reference passing back
	//}
	public inline function len():Int {
		return length;
	}
	public inline function cap():Int {
		return capacity-start;
	}
	public inline function itemAddr(idx:Int):Pointer {
		//if (idx<0 || idx>=(end-start)) Scheduler.panicFromHaxe("Slice index out of range for addr()");
		return baseArray.addr((idx+start)*itemSize);
	}
	public function toString():String {
		var ret:String = "Slice{"+start+","+end+",[";
		if(baseArray!=null) 
			for(i in 0...(start+capacity) ) {
				if(i!=0) ret += ",";
				ret+=baseArray.addr(i*itemSize).toString(itemSize); // only works for basic types
			}
		return ret+"]}";
	}
}

@:keep
class Closure { // "closure" is a keyword in PHP but solved using compiler flag  --php-prefix go  //TODO tidy names
	public var fn:Dynamic; 
	public var bds:Dynamic; // actually an anon struct

	public function new(f:Dynamic,b:Dynamic) {
		if(Std.is(f,Closure))	{
			if(!Reflect.isFunction(f.fn)) Scheduler.panicFromHaxe( "invalid function reference passed to make Closure(): "+f.fn);
			fn=f.fn; 
		} else{
			if(!Reflect.isFunction(f)) Scheduler.panicFromHaxe("invalid function reference passed to make Closure(): "+f); 
	 		fn=f;
		}
		if(fn==null) Scheduler.panicFromHaxe("new Closure() function has become null!"); // error test for flash/cpp TODO remove when issue resolved
		bds=b;
	}
	public function toString():String {
		var ret:String = "Closure{"+fn+",";
		//for(i in 0...bds.length) {
		//	if(i!=0) ret += ",";
		//	ret+= bds[i];
		//}
		return ret+bds.toString()+"}";
	}
	public function methVal(t:Dynamic,v:Dynamic):Dynamic{
		return Reflect.callMethod(null, fn, [[],t,v]);
	}
	public function callFn(params:Dynamic):Dynamic {
		if(fn==null) Scheduler.panicFromHaxe("attempt to call null function reference in Closure()");
		if(!Reflect.isFunction(fn)) Scheduler.panicFromHaxe("invalid function reference in Closure(): "+fn);
		return Reflect.callMethod(null, fn, params);
	}
}

class Interface{ // "interface" is a keyword in PHP but solved using compiler flag  --php-prefix go //TODO tidy names 
	public var typ:Int; // the possibly interface type that has been cast to
	public var val:Dynamic;

	public function new(t:Int,v:Dynamic){
		typ=t;
		val=v; 
	}
	public function toString():String {
		if(val==null)
			return "Interface{nil:"+TypeInfo.getName(typ)+"}";
		else
			return "Interface{"+val+":"+TypeInfo.getName(typ)+"}";
	}
	public static function change(t:Int,i:Interface):Interface {
		if(i==null)	
			if(TypeInfo.isConcrete(t)) 
				return new Interface(t,TypeInfo.zeroValue(t)); 
			else {
				Scheduler.panicFromHaxe( "can't change the Interface of a nil value to Interface type: " +TypeInfo.getName(t));  
				return new Interface(t,TypeInfo.zeroValue(t));	 //dummy value as we have hit the panic button
			}
		else 
			if(Std.is(i,Interface)) 	
				if(TypeInfo.isConcrete(t)) 
					return new Interface(t,i.val); 
				else
					return new Interface(i.typ,i.val); // do not allow non-concrete types for Interfaces
			else {
				Scheduler.panicFromHaxe( "Can't change the Interface of a non-Interface type:"+i+" to: "+TypeInfo.getName(t));  
				return new Interface(t,TypeInfo.zeroValue(t));	 //dummy value as we have hit the panic button
			}
	}
	public static function isEqual(a:Interface,b:Interface):Bool {		// TODO ensure this very wide definition of equality is OK 
		if(a==null) 
			if(b==null) return true;
			else 		return false;
		if(b==null)		
			return false;
		if(! (TypeInfo.isIdentical(a.typ,b.typ)||TypeInfo.isAssignableTo(a.typ,b.typ)||	TypeInfo.isAssignableTo(b.typ,a.typ)) ) 
			return false;
		else
			if(a.val==b.val) 
				return true; // simple equality
			else // could still be equal underneath a pointer    //TODO is another special case required for Slice?
				if(Std.is(a.val,Pointer) && Std.is(b.val,Pointer))
					return a.val.isEqual(b.val);
				else
					return false;	
	}			
	/* from the SSA documentation:
	If AssertedType is a concrete type, TypeAssert checks whether the dynamic type in Interface X is equal to it, and if so, 
		the result of the conversion is a copy of the value in the Interface.
	If AssertedType is an Interface, TypeAssert checks whether the dynamic type of the Interface is assignable to it, 
		and if so, the result of the conversion is a copy of the Interface value X. If AssertedType is a superInterface of X.Type(), 
		the operation will fail iff the operand is nil. (Contrast with ChangeInterface, which performs no nil-check.)
	*/
	public static function assert(assTyp:Int,ifce:Interface):Dynamic{
		if(ifce==null) {
			Scheduler.panicFromHaxe( "Interface.assert null Interface");
			return null;
		}
		if(!(TypeInfo.isAssignableTo(ifce.typ,assTyp)||TypeInfo.isIdentical(assTyp,ifce.typ))) { // TODO review need for isIdentical 
			Scheduler.panicFromHaxe( "type assert failed: expected "+TypeInfo.getName(assTyp)+", got "+TypeInfo.getName(ifce.typ) );
			return null;
		}
		if(TypeInfo.isConcrete(assTyp))	
			return ifce.val;
		else	
			return new Interface(ifce.typ,ifce.val);
	}
	public static function assertOk(assTyp:Int,ifce:Interface):{r0:Dynamic,r1:Bool} {
		if(ifce==null) 
			return {r0:TypeInfo.zeroValue(assTyp),r1:false};
		if(!(TypeInfo.isAssignableTo(ifce.typ,assTyp)||TypeInfo.isIdentical(assTyp,ifce.typ))) // TODO review need for isIdentical 
			return {r0:TypeInfo.zeroValue(assTyp),r1:false};
		if(TypeInfo.isConcrete(assTyp))	
			return {r0:ifce.val,r1:true};
		else	
			return {r0:new Interface(ifce.typ,ifce.val),r1:true};
	}
	public static function invoke(ifce:Interface,meth:String,args:Array<Dynamic>):Dynamic {
		if(ifce==null) 
			Scheduler.panicFromHaxe( "Interface.invoke null Interface"); 
		//trace("Invoke:"+ifce+":"+meth);
		if(!Std.is(ifce,Interface)) 
			Scheduler.panicFromHaxe( "Interface.invoke on non-Interface value"); 
		//return Reflect.callMethod(o:Dynamic, func:Dynamic, args:Array<Dynamic>);
		var fn:Dynamic=TypeInfo.method(ifce.typ,meth);
		//trace("Invoke:"+TypeInfo.getName(ifce.typ)+":"+meth+":"+ifce.val+":"+fn);
		//return fn([],Deep.copy(ifce.val));
		return Reflect.callMethod(null, fn, args);
	}
}

class Channel<T> { //TODO check close & rangeing over a channel
var entries:Array<T>;
var max_entries:Int;
var num_entries:Int;
var oldest_entry:Int;	
var closed:Bool;

public function new(how_many_entries:Int) {
	if(how_many_entries<=0)
		how_many_entries=1;
	entries = new Array<T>();
	max_entries = how_many_entries;
	oldest_entry = 0;
	num_entries = 0;
	closed = false;
}
public function hasSpace():Bool {
	if(this==null) return false; // non-existant channels never have space
	if(closed) return false; // closed channels don't have space
	return num_entries < max_entries;
}
public function send(source:T):Bool {
	if(closed) Scheduler.panicFromHaxe( "attempt to send to closed channel"); 
	var next_element:Int;
	if (this.hasSpace()) {
		next_element = (oldest_entry + num_entries) % max_entries;
		num_entries++;
		entries[next_element]=source;  
		return true;
	} else
		return false;
}
public function hasNoContents():Bool { // used by channel read
	if (this==null) return true; // spec: "Receiving from a nil channel blocks forever."
	if (closed) return false; // spec: "Receiving from a closed channel always succeeds..."
	else return num_entries == 0;
}
public function hasContents():Bool { // used by select
	if (this==null) return false; // spec: "Receiving from a nil channel blocks forever."
	if (closed) return true; // spec: "Receiving from a closed channel always succeeds..."
	return num_entries != 0;
}
public function receive(zero:T):{r0:T ,r1:Bool} {
	var ret:T=zero;
	if (num_entries > 0) {
		ret=entries[oldest_entry];
		oldest_entry = (oldest_entry + 1) % max_entries;
		num_entries--;
		return {r0:ret,r1:true};
	} else
		if(closed)
			return {r0:ret,r1:false}; // spec: "Receiving from a closed channel always succeeds, immediately returning the element type's zero value."
		else {
			Scheduler.panicFromHaxe( "channel receive unreachable code!"); 
			return {r0:ret,r1:false}; //dummy value as we have hit the panic button
		}
}
public inline function len():Int { 
	return num_entries; 
}
public inline function cap():Int { 
	return max_entries; 
}
public inline function close() {
	if(this==null) Scheduler.panicFromHaxe( "attempt to close a nil channel" ); 
	closed = true;
}
}

class Complex {
	public var real:Float;
	public var imag:Float;
public inline function new(r:Float, i:Float) {
	real = r;
	imag = i;
}
public static inline function neg(x:Complex):Complex {
	return new Complex(0.0-x.real,0.0-x.imag);
}
public static inline function add(x:Complex,y:Complex):Complex {
	return new Complex(x.real+y.real,x.imag+y.imag);
}
public static inline function sub(x:Complex,y:Complex):Complex {
	return new Complex(x.real-y.real,x.imag-y.imag);
}
public static inline function mul(x:Complex,y:Complex):Complex {
	return new Complex( (x.real * y.real) - (x.imag * y.imag), (x.imag * y.real) + (x.real * y.imag));
}
public static function div(x:Complex,y:Complex):Complex {
	if( (y.real == 0.0) && (y.imag == 0.0) ){
		Scheduler.panicFromHaxe( "complex divide by zero");
		return new Complex(0.0,0.0); //dummy value as we have hit the panic button
	} else {
		return new Complex(
			((x.real * y.real) + (x.imag * y.imag)) / ((y.real * y.real) + (y.imag * y.imag)) ,
			((x.imag * y.real) - (x.real * y.imag)) / ((y.real * y.real) + (y.imag * y.imag)) );
	}
}
public static inline function eq(x:Complex,y:Complex):Bool { // "=="
	return (x.real == y.real) && (x.imag == y.imag);
}
public static inline function neq(x:Complex,y:Complex):Bool { // "!="
	return (x.real != y.real) || (x.imag != y.imag);
}
}

// optimize to use cs and java base i64 types 
//#if ( cpp || cs || java )
//	typedef HaxeInt64Typedef = haxe.Int64; // these implementations are using native types
//#else
	typedef HaxeInt64Typedef = Int64;  // use the copied and modified version of the standard library class below
//	// TODO revert to haxe.Int64 when the version below (or better) reaches the released libray
//#end

// this abstract type to enable correct handling for Go of HaxeInt64Typedef
abstract HaxeInt64abs(HaxeInt64Typedef) 
from HaxeInt64Typedef to HaxeInt64Typedef 
{ 
inline function new(v:HaxeInt64Typedef) this=v;

public static inline function toInt(v:HaxeInt64abs):Int {

/* TODO re-optimize to use cs and java base i64 types once all working
	#if java 
		return HaxeInt64Typedef.toInt(v); // NOTE: java version just returns low 32 bits
	#else
*/
		return HaxeInt64Typedef.getLow(v); // NOTE: does not throw an error if value overflows Int

/* TODO re-optimize to use cs and java base i64 types once all working
	#end
*/
}
public static inline function ofInt(v:Int):HaxeInt64abs {
	return new HaxeInt64abs(HaxeInt64Typedef.ofInt(v));
}
public static function toFloat(vp:HaxeInt64abs):Float{ // signed int64 to float (TODO auto-cast of Unsigned pos problem)
		//TODO native versions for java & cs
		var v:HaxeInt64Typedef=vp;
		var isNegVal:Bool=false;
		if(isNeg(v)) {
			if(compare(v,make(0x80000000,0))==0) return -9223372036854775808.0; // most -ve value can't be made +ve
			isNegVal=true;
			v=neg(v);	
		}
		var ret:Float=0.0;
		var multiplier:Float=1.0;
		var one:HaxeInt64abs=make(0,1);
		for(i in 0...63) { // TODO improve speed by calculating more than 1 bit at a time
			if(!isZero(and(v,one)))
				ret += multiplier;
			multiplier *= 2.0;
			v=ushr(v,1);
		}
		if(isNegVal) return -ret;
		return ret;
}
public static function toUFloat(vp:HaxeInt64abs):Float{ // unsigned int64 to float
		//TODO native versions for java & cs
		var v:HaxeInt64Typedef=vp;
		var ret:Float=0.0;
		var multiplier:Float=1.0;
		var one:HaxeInt64abs=make(0,1);
		for(i in 0...64) { // TODO improve speed by calculating more than 1 bit at a time
			if(!isZero(and(v,one)))
	 			ret += multiplier;
			multiplier *= 2.0;
			v=ushr(v,1);
		}
		return ret;
}
public static function ofFloat(v):HaxeInt64abs { // float to signed int64 (TODO auto-cast of Unsigned is a posible problem)
		//TODO native versions for java & cs
		if(v==0.0) return make(0,0); 
		if(Math.isNaN(v)) return make(0x80000000,0); // largest -ve number is returned by Go in this situation
		var isNegVal:Bool=false;
		if(v<0.0){
			isNegVal=true;
			v = -v;
		} 
		if(v<2147483647.0) { // optimization: if just a small integer, don't do the full conversion code below
			if(isNegVal) 	return new HaxeInt64abs(HaxeInt64Typedef.neg(HaxeInt64Typedef.ofInt(Math.ceil(v))));
			else			return new HaxeInt64abs(HaxeInt64Typedef.ofInt(Math.floor(v)));
		}
		if(v>9223372036854775807.0) { // number too big to encode in 63 bits 
			if(isNegVal)	return new HaxeInt64abs(HaxeInt64Typedef.make(0x80000000,0)); 			// largest -ve number
			else			return new HaxeInt64abs(HaxeInt64Typedef.make(0x7fffffff,0xffffffff)); 	// largest +ve number
		}
		var f32:Float = 4294967296.0 ; // the number of combinations in 32-bits
		var f16:Float = 65536.0; // the number of combinations in 16-bits
		var high:Int = Math.floor(v/f32); 
		var lowFloat:Float= Math.ffloor(v-(high*f32)) ;
		var lowTop16:Int = Math.floor(lowFloat/f16) ;
		var lowBot16:Int = Math.floor(lowFloat-(lowTop16*f16)) ;
		var res:HaxeInt64Typedef = HaxeInt64Typedef.make(high,lowBot16);
		res = HaxeInt64Typedef.or(res,HaxeInt64Typedef.shl(HaxeInt64Typedef.make(0,lowTop16),16));
		if(isNegVal) return new HaxeInt64abs(HaxeInt64Typedef.neg(res));
		return new HaxeInt64abs(res);
}
public static function ofUFloat(v):HaxeInt64abs { // float to un-signed int64 
		//TODO native versions for java & cs
		if(v<=0.0) return make(0,0); // -ve values are invalid, so return 0
		if(Math.isNaN(v)) return make(0x80000000,0); // largest -ve number is returned by Go in this situation
		if(v<2147483647.0) { // optimization: if just a small integer, don't do the full conversion code below
			return ofInt(Math.floor(v));
		}
		if(v>18446744073709551615.0) { // number too big to encode in 64 bits 
			return new HaxeInt64abs(HaxeInt64Typedef.make(0xffffffff,0xffffffff)); 	// largest unsigned number
		}
		var f32:Float = 4294967296.0 ; // the number of combinations in 32-bits
		var f16:Float = 65536.0; // the number of combinations in 16-bits
		var high:Int = Math.floor(v/f32); 
		var lowFloat:Float= Math.ffloor(v-(high*f32)) ;
		var lowTop16:Int = Math.floor(lowFloat/f16) ;
		var lowBot16:Int = Math.floor(lowFloat-(lowTop16*f16)) ;
		var res:HaxeInt64Typedef = HaxeInt64Typedef.make(high,lowBot16);
		res = HaxeInt64Typedef.or(res,HaxeInt64Typedef.shl(HaxeInt64Typedef.make(0,lowTop16),16));
		return new HaxeInt64abs(res);
}
public static inline function make(h:Int,l:Int):HaxeInt64abs {
		return new HaxeInt64abs(HaxeInt64Typedef.make(h,l));
}
public static inline function toString(v:HaxeInt64abs):String {
	return HaxeInt64Typedef.toStr(v);
}
public static inline function toStr(v:HaxeInt64abs):String {
	return HaxeInt64Typedef.toStr(v);
}
public static inline function neg(v:HaxeInt64abs):HaxeInt64abs {
	return new HaxeInt64abs(HaxeInt64Typedef.neg(v));
}
public static inline function isZero(v:HaxeInt64abs):Bool {
	return HaxeInt64Typedef.isZero(v);
}
public static inline function isNeg(v:HaxeInt64abs):Bool {
	return HaxeInt64Typedef.isNeg(v);
}
public static inline function add(x:HaxeInt64abs,y:HaxeInt64abs):HaxeInt64abs {
	return new HaxeInt64abs(HaxeInt64Typedef.add(x,y));
}
public static inline function and(x:HaxeInt64abs,y:HaxeInt64abs):HaxeInt64abs {
	return new HaxeInt64abs(HaxeInt64Typedef.and(x,y));
}
private static function checkDiv(x:HaxeInt64abs,y:HaxeInt64abs,isSigned:Bool):HaxeInt64abs {
	if(HaxeInt64Typedef.isZero(y))
		Scheduler.panicFromHaxe( "attempt to divide 64-bit value by 0"); 
	if(isSigned && (HaxeInt64Typedef.compare(y,HaxeInt64Typedef.ofInt(-1))==0) && (HaxeInt64Typedef.compare(x,HaxeInt64Typedef.make(0x80000000,0))==0) ) 
	{
		//trace("checkDiv 64-bit special case");
		y=HaxeInt64Typedef.ofInt(1); // special case in the Go spec
	}
	return new HaxeInt64abs(y);
}
public static function div(x:HaxeInt64abs,y:HaxeInt64abs,isSigned:Bool):HaxeInt64abs {
	y=checkDiv(x,y,isSigned);
	if(HaxeInt64Typedef.compare(y,HaxeInt64Typedef.ofInt(1))==0) return new HaxeInt64abs(x);
	if(isSigned || (!HaxeInt64Typedef.isNeg(x) && !HaxeInt64Typedef.isNeg(y)))
		return new HaxeInt64abs(HaxeInt64Typedef.div(x,y));
	else {
		if(	HaxeInt64Typedef.isNeg(x) ) {
			if( HaxeInt64Typedef.isNeg(y) ){ // both x and y are "-ve""
				if( HaxeInt64Typedef.compare(x,y) < 0 ) { // x is more "-ve" than y, so the smaller uint   
					return new HaxeInt64abs(HaxeInt64Typedef.ofInt(0));						
				} else {
					return new HaxeInt64abs(HaxeInt64Typedef.ofInt(1));	
				}
			} else { // only x is -ve
				var pt1:HaxeInt64Typedef = HaxeInt64Typedef.make(0x7FFFFFFF,0xFFFFFFFF); // the largest part of the numerator
				var pt2:HaxeInt64Typedef = HaxeInt64Typedef.and(x,pt1); // the smaller part of the numerator
				var rem:HaxeInt64Typedef = HaxeInt64Typedef.make(0,1); // the left-over bit
				rem = HaxeInt64Typedef.add(rem,HaxeInt64Typedef.mod(pt1,y));
				rem = HaxeInt64Typedef.add(rem,HaxeInt64Typedef.mod(pt2,y));
				if( HaxeInt64Typedef.ucompare(rem,y) >= 0 ) { // the remainder is >= divisor  
					rem = HaxeInt64Typedef.ofInt(1);
				} else {
					rem = HaxeInt64Typedef.ofInt(0);
				}
				pt1 = HaxeInt64Typedef.div(pt1,y);	
				pt2 = HaxeInt64Typedef.div(pt2,y);			
				return new HaxeInt64abs(HaxeInt64Typedef.add(pt1,HaxeInt64Typedef.add(pt2,rem)));	
			}
		}else{ // logically, y is "-ve"" but x is "+ve" so y>x , so any divide will yeild 0
				return new HaxeInt64abs(HaxeInt64Typedef.ofInt(0));	
		}
	}
}
public static function mod(x:HaxeInt64abs,y:HaxeInt64abs,isSigned:Bool):HaxeInt64abs {
	y=checkDiv(x,y,isSigned);
	if(HaxeInt64Typedef.compare(y,HaxeInt64Typedef.ofInt(1))==0) return new HaxeInt64abs(HaxeInt64Typedef.ofInt(0));
	return new HaxeInt64abs(HaxeInt64Typedef.mod(x,y));
}
public static inline function mul(x:HaxeInt64abs,y:HaxeInt64abs):HaxeInt64abs {
	return new HaxeInt64abs(HaxeInt64Typedef.mul(x,y));
}
public static inline function or(x:HaxeInt64abs,y:HaxeInt64abs):HaxeInt64abs {
	return new HaxeInt64abs(HaxeInt64Typedef.or(x,y));
}
public static inline function shl(x:HaxeInt64abs,y:Int):HaxeInt64abs {
	if(y==64) // this amount of shl is not handled correcty by the underlying code
		return new HaxeInt64abs(HaxeInt64Typedef.ofInt(0));	
	else
		return new HaxeInt64abs(HaxeInt64Typedef.shl(x,y));
}
public static inline function shr(x:HaxeInt64abs,y:Int):HaxeInt64abs {
	return new HaxeInt64abs(HaxeInt64Typedef.shr(x,y));
}
public static function ushr(x:HaxeInt64abs,y:Int):HaxeInt64abs { // note, not inline
	#if php
	if(y==32){ // error with php on 32 bit right shift for uint64, so do 2x16
		var ret:HaxeInt64Typedef = HaxeInt64Typedef.ushr(x,16);
		return new HaxeInt64abs(HaxeInt64Typedef.ushr(ret,16));
	}
	#end
	return new HaxeInt64abs(HaxeInt64Typedef.ushr(x,y));
}
public static inline function sub(x:HaxeInt64abs,y:HaxeInt64abs):HaxeInt64abs {
	return new HaxeInt64abs(HaxeInt64Typedef.sub(x,y));
}
public static inline function xor(x:HaxeInt64abs,y:HaxeInt64abs):HaxeInt64abs {
	return new HaxeInt64abs(HaxeInt64Typedef.xor(x,y));
}
public static inline function compare(x:HaxeInt64abs,y:HaxeInt64abs):Int {
	return HaxeInt64Typedef.compare(x,y);
}
public static function ucompare(x:HaxeInt64abs,y:HaxeInt64abs):Int {
	#if ( java || cs )
		// unsigned compare library code does not work properly for these platforms 
		if(HaxeInt64Typedef.isZero(x)) {
			if(HaxeInt64Typedef.isZero(y)) {
				return 0;
			} else {
				return -1; // any value is larger than x 
			}
		}
		if(HaxeInt64Typedef.isZero(y)) { // if we are here, we know that x is non-zero
				return 1; // any value of x is larger than y 
		}
		if(!HaxeInt64Typedef.isNeg(x)) { // x +ve
			if(!HaxeInt64Typedef.isNeg(y)){ // both +ve so normal comparison
				return HaxeInt64Typedef.compare(x,y);
			}else{ // y -ve and so larger than x
				return -1;
			}
		}else { // x -ve
			if(!HaxeInt64Typedef.isNeg(y)){ // -ve x larger than +ve y
				return 1;
			}else{ // both are -ve so the normal comparison
				return HaxeInt64Typedef.compare(x,y); //eg -1::-7 gives -1--7 = +6 meaning -1 > -7 which is correct for unsigned
			}
		}
	#else
	 	return HaxeInt64Typedef.ucompare(x,y);
	#end
}
}

	typedef GOint64 = HaxeInt64abs;
/* TODO re-optimize to use cs and java base i64 types once all working
#end
*/

//**************** rewrite of std Haxe library function haxe.Int64 for PHP integer overflow an other errors
/*
Modify haxe.Int64.hx to work on php and fix other errors
- php integer overflow and ushr are incorrect (for 32-bits Int),
special functions now correct for these faults for Int64.
- both div and mod now have the sign correct when double-negative.
- special cases of div or mod by 0 or 1 now correct.
*/
/*
 * Copyright (C)2005-2012 Haxe Foundation
 *
 * Permission is hereby granted, free of charge, to any person obtaining a
 * copy of this software and associated documentation files (the "Software"),
 * to deal in the Software without restriction, including without limitation
 * the rights to use, copy, modify, merge, publish, distribute, sublicense,
 * and/or sell copies of the Software, and to permit persons to whom the
 * Software is furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
 * FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
 * DEALINGS IN THE SOFTWARE.
 */
class Int64 { 

	var high : Int;
	var low : Int;

	inline function new(high, low) {
		this.high = i32(high);
		this.low = i32(low);
	}

	#if php
	/*
		private function to correctly handle 32-bit integer overflow on php 
		see: http://stackoverflow.com/questions/300840/force-php-integer-overflow
	*/
	private static function i32php(value:Int):Int { 
			value = (value & untyped __php__("0xFFFFFFFF"));
 		    if ( (value & untyped __php__("0x80000000"))!=0 )
		        value = -(((~value) & untyped __php__("0xFFFFFFFF")) + 1);
		    return value;
	}
	#end

	/*
		private function to correctly handle 32-bit ushr on php
		see: https://github.com/HaxeFoundation/haxe/commit/1a878aa90708040a41b0dd59f518d83b09ede209
	*/
	private static inline function ushr32(v:Int,n:Int):Int { 
		#if php
		 	return (v >> n) & (untyped __php__("0x7fffffff") >> (n-1));
		#else
			return v>>>n;
		#end
	}

	@:extern static inline function i32(i) {
		#if (js || flash8)
			return i | 0;
		#elseif php
			return i32php(i); // handle overflow of 32-bit integers correctly 
		#else
			return i;
		#end
	}

	@:extern static inline function i32mul(a:Int,b:Int) {
		#if (php || js || flash8)
		/*
			We can't simply use i32(a*b) since we might overflow (52 bits precision in doubles)
		*/
		return i32(i32((a * (b >>> 16)) << 16) + (a * (b&0xFFFF)));
		#else
		return a * b;
		#end
	}
	
	#if as3 public #end function toString() {
		if ((high|low) == 0 )
			return "0";
		var str = "";
		var neg = false;
		var i = this;
		if( isNeg(i) ) {
			neg = true;
			i = Int64.neg(i);
		}
		var ten = ofInt(10);
		while( !isZero(i) ) {
			var r = divMod(i, ten);
			str = r.modulus.low + str; 
			i = r.quotient; 
		}
		if( neg ) str = "-" + str;
		return str;
	}

	public static inline function make( high : Int, low : Int ) : Int64 {
		return new Int64(high, low); 
	}

	public static inline function ofInt( x : Int ) : Int64 {
		return new Int64(x >> 31,x);
	}

	public static function toInt( x : Int64 ) : Int {
		if( x.high != 0 ) {
			if( x.high < 0 )
				return -toInt(neg(x));
			throw "Overflow"; //NOTE go panic not used here as it is in the Haxe libary code
		}
		return x.low; 
	}

	public static function getLow( x : Int64 ) : Int {
		return x.low;
	}

	public static function getHigh( x : Int64 ) : Int {
		return x.high;
	}

	public static function add( a : Int64, b : Int64 ) : Int64 {
		var high = i32(a.high + b.high);
		var low = i32(a.low + b.low);
		if( uicompare(low,a.low) < 0 )
			high++;
		return new Int64(high, low);
	}

	public static function sub( a : Int64, b : Int64 ) : Int64 {
		var high = i32(a.high - b.high); // i32() call required to match add
		var low = i32(a.low - b.low); // i32() call required to match add
		if( uicompare(a.low,b.low) < 0 )
			high--;
		return new Int64(high, low);
	}

	public static function mul( a : Int64, b : Int64 ) : Int64 {
		var mask = 0xFFFF;
		var al = a.low & mask, ah = ushr32(a.low , 16); 
		var bl = b.low & mask, bh = ushr32(b.low , 16); 
		var p00 = al * bl;
		var p10 = ah * bl;
		var p01 = al * bh;
		var p11 = ah * bh;
		var low = p00;
		var high = i32(p11 + ushr32(p01 , 16) + ushr32(p10 , 16));
		p01 = i32(p01 << 16); low = i32(low + p01); if( uicompare(low, p01) < 0 ) high = i32(high + 1);
		p10 = i32(p10 << 16); low = i32(low + p10); if( uicompare(low, p10) < 0 ) high = i32(high + 1);
		high = i32(high + i32mul(a.low,b.high));
		high = i32(high + i32mul(a.high,b.low));
		return new Int64(high, low);
	}

	static function divMod( modulus : Int64, divisor : Int64 ) {
		var quotient = new Int64(0, 0);
		var mask = new Int64(0, 1);
		divisor = new Int64(divisor.high, divisor.low);
		while( divisor.high >= 0 ) { 
			var cmp = ucompare(divisor, modulus);
			divisor.high = i32( i32(divisor.high << 1) | ushr32(divisor.low , 31) ); 
			divisor.low = i32(divisor.low << 1); 
			mask.high = i32( i32(mask.high << 1) | ushr32(mask.low , 31) ); 
			mask.low = i32(mask.low << 1);
			if( cmp >= 0 ) break;
		}
		while( i32(mask.low | mask.high) != 0 ) { 
			if( ucompare(modulus, divisor) >= 0 ) {
				quotient.high= i32(quotient.high | mask.high); 
				quotient.low= i32(quotient.low | mask.low); 
				modulus = sub(modulus,divisor);
			}
			mask.low = i32( ushr32(mask.low , 1) | i32(mask.high << 31) ); 
			mask.high = ushr32(mask.high , 1); 

			divisor.low = i32( ushr32(divisor.low , 1) | i32(divisor.high << 31) ); 
			divisor.high = ushr32(divisor.high , 1); 
		}
		return { quotient : quotient, modulus : modulus };
	}

	public static function div( a : Int64, b : Int64 ) : Int64 { 
		if(b.high==0) // handle special cases of 0 and 1
			switch(b.low) {
			case 0:	throw "divide by zero";  //NOTE go panic not used here as it is in the Haxe libary code
			case 1: return new Int64(a.high,a.low);
			} 
		var sign = ((a.high<0) || (b.high<0)) && (!( (a.high<0) && (b.high<0))); // make sure we get the correct sign
		if( a.high < 0 ) a = neg(a);
		if( b.high < 0 ) b = neg(b);
		var q = divMod(a, b).quotient;
		return sign ? neg(q) : q;
	}

	public static function mod( a : Int64, b : Int64 ) : Int64 {
		if(b.high==0) // handle special cases of 0 and 1
			switch(b.low) {
			case 0:	throw "modulus by zero";  //NOTE go panic not used here as it is in the Haxe libary code
			case 1: return ofInt(0);
			}
		var sign = a.high<0; // the sign of a modulus is the sign of the value being mod'ed
		if( a.high < 0 ) a = neg(a);
		if( b.high < 0 ) b = neg(b);
		var m = divMod(a, b).modulus;
		return sign ? neg(m) : m;
	}

	public static inline function shl( a : Int64, b : Int ) : Int64 {
		return if( b & 63 == 0 ) a else if( b & 63 < 32 ) new Int64( (a.high << b) | ushr32(a.low, i32(32-(b&63))), a.low << b ) else new Int64( a.low << i32(b - 32), 0 );
	}

	public static inline function shr( a : Int64, b : Int ) : Int64 {
		return if( b & 63 == 0 ) a else if( b & 63 < 32 ) new Int64( a.high >> b, ushr32(a.low,b) | (a.high << i32(32 - (b&63))) ) else new Int64( a.high >> 31, a.high >> i32(b - 32) );
	}

	public static inline function ushr( a : Int64, b : Int ) : Int64 {
		return if( b & 63 == 0 ) a else if( b & 63 < 32 ) new Int64( ushr32(a.high, b), ushr32(a.low, b) | (a.high << i32(32 - (b&63))) ) else new Int64( 0, ushr32(a.high, i32(b - 32)) );
	}

	public static inline function and( a : Int64, b : Int64 ) : Int64 {
		return new Int64( a.high & b.high, a.low & b.low );
	}

	public static inline function or( a : Int64, b : Int64 ) : Int64 {
		return new Int64( a.high | b.high, a.low | b.low );
	}

	public static inline function xor( a : Int64, b : Int64 ) : Int64 {
		return new Int64( a.high ^ b.high, a.low ^ b.low );
	}

	public static inline function neg( a : Int64 ) : Int64 {
		var high = i32(~a.high); 
		var low = i32(-a.low); 
		if( low == 0 )
			high++;
		return new Int64(high,low);
	}

	public static inline function isNeg( a : Int64 ) : Bool {
		return a.high < 0;
	}

	public static inline function isZero( a : Int64 ) : Bool {
		return (a.high | a.low) == 0;
	}

	static function uicompare( a : Int, b : Int ) {
		return a < 0 ? (b < 0 ? i32(~b - ~a) : 1) : (b < 0 ? -1 : i32(a - b));
	}

	public static inline function compare( a : Int64, b : Int64 ) : Int {
		var v = i32(a.high - b.high); 
		return if( v != 0 ) v else uicompare(a.low,b.low);
	}

	/**
		Compare two Int64 in unsigned mode.
	**/
	public static inline function ucompare( a : Int64, b : Int64 ) : Int {
		var v = uicompare(a.high,b.high);
		return if( v != 0 ) v else uicompare(a.low, b.low);
	}

	public static inline function toStr( a : Int64 ) : String {
		return a.toString();
	}

}
//**************** END REWRITE of haxe.Int64 for php and to correct errors


// GoRoutine 
class StackFrameBasis
{
public var _incomplete(default,null):Bool=true;
public var _latestPH:Int=0;
public var _latestBlock:Int=0;
public var _functionPH:Int;
public var _functionName:String;
public var _goroutine(default,null):Int;
public var _bds:Dynamic; // bindings for closures
public var _deferStack:List<StackFrame>;

public function new(gr:Int,ph:Int,name:String){
	_goroutine=gr;
	_functionPH=ph;
	_functionName=name;
	_deferStack=new List<StackFrame>();
	// TODO optionally profile function entry here
}

public function setLatest(ph:Int,blk:Int){ // this can be done inline, but generates too much code
	this.setPH(ph);
	_latestBlock=blk;
	// TODO optionally profile block entry here
}

public function setPH(ph:Int){
	_latestPH=ph;
	// TODO optionally profile instruction line entry here	
}

public inline function defer(fn:StackFrame){
	//trace("defer");
	_deferStack.add(fn); // add to the end of the list, so that runDefers() get them in the right order
}

public function runDefers(){
	//trace("runDefers");
	while(!_deferStack.isEmpty()){
		//trace("runDefers-pop");
		Scheduler.push(_goroutine,_deferStack.pop());
		//Scheduler.traceStackDump();
	}
}


}

interface StackFrame
{
public var _incomplete(default,null):Bool;
public var _latestPH:Int;
public var _latestBlock:Int;
public var _functionPH:Int;
public var _functionName:String;
public var _goroutine(default,null):Int;
public var _bds:Dynamic; // bindings for closures as a anonymous struct
public var _deferStack:List<StackFrame>;
function run():StackFrame; // function state machine (set up by each Go function Haxe class)
function res():Dynamic; // function result (set up by each Go function Haxe class)
}

class Scheduler { // NOTE this code requires a single-thread, as there is no locking 
// public
public static var doneInit:Bool=false; // flag to limit go-routines to 1 during the init() processing phase
// private
static var grStacks:Array<List<StackFrame>>=new Array<List<StackFrame>>(); 
static var grInPanic:Array<Bool>=new Array<Bool>();
static var grPanicMsg:Array<Interface>=new Array<Interface>();
static var panicStackDump:String="";
static var entryCount:Int=0; // this to be able to monitor the re-entrys into this routine for debug
static var currentGR:Int=0; // the current goroutine, used by Scheduler.panicFromHaxe(), NOTE this requires a single thread

public static function timerEventHandler(dummy:Dynamic) { // if the scheduler is being run from a timer, this is where it comes to
	runAll();
}

public static function runAll() { // this must be re-entrant, in order to allow Haxe->Go->Haxe->Go for some runtime functions
	var cg:Int=0; // reentrant current goroutine
	entryCount++;
	if(entryCount>2) { // this is the simple limit to runtime recursion  
		throw "Scheduler.runAll() entryCount exceeded - "+stackDump();
	}

	// special handling for goroutine 0, which is used in the initialisation phase and re-entrantly, where only one goroutine may operate		
	if(grStacks[0].isEmpty()) { // check if there is ever likley to be anything to do
		if(grStacks.length<=1) { 
			throw "Scheduler: there is only one goroutine and its stack is empty\n"+stackDump();		
		}
	} else { // run goroutine zero
		runOne(0,entryCount);
	}

	if(doneInit  && entryCount==1 ) {	// don't run extra goroutines when we are re-entrant or have not finished initialistion
									// NOTE this means that Haxe->Go->Haxe->Go code cannot run goroutines 
		for(cg in 1...grStacks.length) { // length may grow during a run through, NOTE goroutine 0 not run again
			if(!grStacks[cg].isEmpty()) {
				runOne(cg,entryCount);
			}
		}
		// prune the list of goroutines only at the end (goroutine numbers are in the stack frames, so can't be altered) 
		while(grStacks.length>1){
			if(grStacks[grStacks.length-1].isEmpty())
				grStacks.pop();
			else
				break;
		}
	}
	entryCount--;
}
static inline function runOne(gr:Int,entryCount:Int){ // called from above to call individual goroutines TODO: Review for multi-threading
	if(grInPanic[gr]) {
		if(entryCount!=1) { // we are in re-entrant code, so we can't panic again, as this may be part of the panic handling...
				// NOTE this means that Haxe->Go->Haxe->Go code cannot use panic() reliably 
				run1(gr);
		} else {
			while(grInPanic[gr]){
				if(grStacks[gr].isEmpty())
					throw "Panic in goroutine "+gr+"\n"+panicStackDump; // use stored stack dump
				else {
					var sf:StackFrame=grStacks[gr].pop();
					while(!sf._deferStack.isEmpty()){ 
						// NOTE this will run all of the defered code for a function, even if recover() is encountered
						// TODO go back to recover code block in SSA function struct after a recover
						var def:StackFrame=sf._deferStack.pop();
						Scheduler.push(gr,def);
						while(def._incomplete) 
							runAll(); // with entryCount >1, so run as above 
					}
				}
			}
		}
	} else {
		run1(gr);
	}
}
public static inline function run1(gr:Int){ // used by callFromRT() for every go function
		if(grStacks[gr].first()==null) { 
			throw "Panic:"+grPanicMsg+"\nScheduler: null stack entry for goroutine "+gr+"\n"+stackDump();
		} else {
			currentGR=gr;
			grStacks[gr].first().run(); // run() may call haxe which calls these routines recursively 
		}	
}
public static function makeGoroutine():Int {
	for (r in 0 ... grStacks.length)
		if(grStacks[r].isEmpty())
		{
			grInPanic[r]=false;
			grPanicMsg[r]=null;
			return r;	// reuse a previous goroutine number if possible
		}
	var l:Int=grStacks.length;
	grStacks[l]=new List<StackFrame>();
	grInPanic[l]=false;
	grPanicMsg[l]=null;
	return l;
}
public static function pop(gr:Int):StackFrame {
	if(gr>=grStacks.length||gr<0)
		throw "Scheduler.pop() invalid goroutine";
	return grStacks[gr].pop();
}
public static function push(gr:Int,sf:StackFrame){
	if(gr>=grStacks.length||gr<0)
		throw "Scheduler.push() invalid goroutine";
	grStacks[gr].push(sf);
}
public static inline function NumGoroutine():Int {
	return grStacks.length;
}

public static function stackDump():String {
	var ret:String = "";
	var gr:Int;
	ret += "runAll() entryCount="+entryCount+"\n";
	for(gr in 0...grStacks.length) {
		ret += "Goroutine " + gr + " "+grPanicMsg[gr]+"\n"; //may need to unpack the interface
		if(grStacks[gr].isEmpty()) {
			ret += "Stack is empty\n";
		} else {
			ret += "Stack has " +grStacks[gr].length+ " entries:\n";
			var it=grStacks[gr].iterator();
			while(it.hasNext()) {
				var ent=it.next();
				if(ent==null) {
					ret += "\tStack entry is null\n";
				} else {
					ret += "\t"+ent._functionName+" starting at "+Go.CPos(ent._functionPH);
					ret += " latest position "+Go.CPos(ent._latestPH);
					ret += " latest block "+ent._latestBlock+"\n";
				}
			}
		}
	}
	return ret;
}

public static function traceStackDump() {trace(stackDump());}

public static function panic(gr:Int,err:Interface){
	if(gr>=grStacks.length||gr<0)
		throw "Scheduler.panic() invalid goroutine";
	if(!grInPanic[gr]) { // if we are already in a panic, keep the first message and stack-dump
		grInPanic[gr]=true;
		grPanicMsg[gr]=err;
		panicStackDump=stackDump();
	}
}
public static function recover(gr:Int):Interface{
	if(gr>=grStacks.length||gr<0)
		throw "Scheduler.recover() invalid goroutine";
	grInPanic[gr]=false;
	return grPanicMsg[gr];
}
public static function panicFromHaxe(err:String) { 
	if(currentGR>=grStacks.length||currentGR<0) 
		// if currnent goroutine is -ve, or out of range, always panics in goroutine 0
		panic(0,new Interface(TypeInfo.getId("string"),"Runtime panic, unknown goroutine, "+err+" "));
	else
		panic(currentGR,new Interface(TypeInfo.getId("string"),"Runtime panic, "+err+" "));
	throw panicStackDump;
}
public static function bbi() {
	panicFromHaxe("bad block ID (internal phi error)");
}
public static function ioor() {
	panicFromHaxe("index out of range");
}
public static inline function wraprangechk(val:Int,sz:Int) {
	if((val<0)||(val>=sz)) ioor();
}
static function unp() {
		panicFromHaxe("unexpected nil pointer (ssa:wrapnilchk)");	
}
public static inline function wrapnilchk(p:Pointer):Pointer {
	if(p==null) unp();
	return p;
}
}


`

/***************************** TODO consider re-using this code to point to general Haxe types ****

@:keep
class Pointer { // slow! heavy use of Dynamic typing and reflection
	private var heapObj:Dynamic; // the actual object holding the value
	private var offs:Array<Int>; // the offsets into the object, if any

	public function new(from:Dynamic){
		heapObj = from; // new Object(from);
		offs = new Array<Int>();
	}
	public function load():Dynamic { // this returns the thing pointed at
		var ret:Dynamic = heapObj;
		for(i in 0...offs.length) {
			try	ret = ret[offs[i]]
			catch (ret:Dynamic)	Scheduler.panicFromHaxe("failed attempt to dereference pointer reading from index:"+offs.toString());
		}
		return ret;
	}
	public function store(v:Dynamic):Void {
		if(offs.length==0)
			heapObj = v;
		else {
			var a:Dynamic = heapObj;
			for(i in 0...offs.length-1) {
				try a = a[offs[i]]
				catch (a:Dynamic) Scheduler.panicFromHaxe("failed attempt to dereference pointer writing to index:"+offs.toString());
			}
			try a[offs[offs.length-1]] = v
			catch (a:Dynamic) Scheduler.panicFromHaxe("failed attempt to dereference pointer writing to index:"+offs.toString());
		}
	}
	public function addr(off:Int):Pointer {
		var ret:Pointer = new Pointer(this.heapObj);
		ret.offs = this.offs.copy();
		ret.offs[this.offs.length]=off;
		return ret;
	}
	public function fieldAddr(f:Int):Pointer {
		var ret:Pointer = new Pointer(this.heapObj);
		ret.offs = this.offs.copy();
		ret.offs[this.offs.length]=f;
		return ret;
	}
	public inline function copy():Pointer {
		return this;
	}
	public inline function load_object(size:Int):Dynamic{
		return this.load();
	}
	public inline function load_bool():Bool {
		return this.load();
	}
	public inline function load_int8():Int {
		return this.load();
	}
	public inline function load_int16():Int {
		return this.load();
	}
	public inline function load_int32():Int {
		return this.load();
	}
	public inline function load_int64():GOint64 {
		return this.load();
	}
	public inline function load_uint8():Int {
		return this.load();
	}
	public inline function load_uint16():Int {
		return this.load();
	}
	public inline function load_uint32():Int {
		return this.load();
	}
	public inline function load_uint64():GOint64 {
		return this.load();
	}
	public inline function load_uintptr():Dynamic { return this.load(); }
	public inline function load_float32():Float {
		return this.load();
	}
	public inline function load_float64():Float {
		return this.load();
	}
	public inline function load_complex64():Complex {
		return this.load();
	}
	public inline function load_complex128():Complex {
		return this.load();
	}
	public inline function load_string():String {
		return this.load();
	}
	public inline function store_object(sz:Int,v:Dynamic):Void { this.store(v); }
	public inline function store_bool(v:Bool):Void { this.store(v); }
	public inline function store_int8(v:Int):Void { this.store(v); }
	public inline function store_int16(v:Int):Void { this.store(v); }
	public inline function store_int32(v:Int):Void { this.store(v); }
	public inline function store_int64(v:GOint64):Void { this.store(v); }
	public inline function store_uint8(v:Int):Void { this.store(v); }
	public inline function store_uint16(v:Int):Void { this.store(v); }
	public inline function store_uint32(v:Int):Void { this.store(v); }
	public inline function store_uint64(v:GOint64):Void { this.store(v); }
	public inline function store_uintptr(v:Dynamic):Void { this.store(v); }
	public inline function store_float32(v:Float):Void { this.store(v); }
	public inline function store_float64(v:Float):Void { this.store(v); }
	public inline function store_complex64(v:Complex):Void { this.store(v); }
	public inline function store_complex128(v:Complex):Void { this.store(v); }
	public inline function store_string(v:String):Void { this.store(v); }
}
*** END pointer for general Haxe types *********************/

/* TODO re-optimize to use cs and java base i64 types once all working
#if ( java || cs )
// this class required to allow load/save of this type via pointer class in Java, as lib fn casts Dynamic to Int64 via Int
// also required in c# to avoid integer overflow errors, probably because of a related problem
// TODO consider ways to optimize

class GOint64  {
private var i64:HaxeInt64abs;

private inline function new(v:HaxeInt64abs) {
	i64=v;
}
public inline function toString():String {
	return HaxeInt64abs.toStr(i64);
}
public static inline function make(h:Int,l:Int):GOint64 {
	return new GOint64(HaxeInt64abs.make(h,l));
}
public static inline function toInt(v:GOint64):Int {
	return HaxeInt64abs.toInt(v.i64);
}
public static inline function toFloat(v:GOint64):Float{
	return HaxeInt64abs.toFloat(v.i64);
}
public static inline function toUFloat(v:GOint64):Float{
	return HaxeInt64abs.toUFloat(v.i64);
}
public static inline function toStr(v:GOint64):String {
	return HaxeInt64abs.toStr(v.i64);
}
public static inline function ofInt(v:Int):GOint64 {
	return new GOint64(HaxeInt64abs.ofInt(v));
}
public static inline function ofFloat(v:Float):GOint64 {
	return new GOint64(HaxeInt64abs.ofFloat(v));
}
public static inline function ofUFloat(v:Float):GOint64 {
	return new GOint64(HaxeInt64abs.ofUFloat(v));
}
public static inline function neg(v:GOint64):GOint64 {
	return new GOint64(HaxeInt64abs.neg(v.i64));
}
public static inline function isZero(v:GOint64):Bool {
	return HaxeInt64abs.isZero(v.i64);
}
public static inline function isNeg(v:GOint64):Bool {
	return HaxeInt64abs.isNeg(v.i64);
}
public static inline function add(x:GOint64,y:GOint64):GOint64 {
	return new GOint64(HaxeInt64abs.add(x.i64,y.i64));
}
public static inline function and(x:GOint64,y:GOint64):GOint64 {
	return new GOint64(HaxeInt64abs.and(x.i64,y.i64));
}
public static inline function div(x:GOint64,y:GOint64,isSigned:Bool):GOint64 {
	return new GOint64(HaxeInt64abs.div(x.i64,y.i64,isSigned));
}
public static inline function mod(x:GOint64,y:GOint64,isSigned:Bool):GOint64 {
	return new GOint64(HaxeInt64abs.mod(x.i64,y.i64,isSigned));
}
public static inline function mul(x:GOint64,y:GOint64):GOint64 {
	return new GOint64(HaxeInt64abs.mul(x.i64,y.i64));
}
public static inline function or(x:GOint64,y:GOint64):GOint64 {
	return new GOint64(HaxeInt64abs.or(x.i64,y.i64));
}
public static inline function shl(x:GOint64,y:Int):GOint64 {
	return new GOint64(HaxeInt64abs.shl(x.i64,y));
}
public static inline function ushr(x:GOint64,y:Int):GOint64 {
	return new GOint64(HaxeInt64abs.ushr(x.i64,y));
}
public static inline function shr(x:GOint64,y:Int):GOint64 {
	return new GOint64(HaxeInt64abs.shr(x.i64,y));
}
public static inline function sub(x:GOint64,y:GOint64):GOint64 {
	return new GOint64(HaxeInt64abs.sub(x.i64,y.i64));
}
public static inline function xor(x:GOint64,y:GOint64):GOint64 {
	return new GOint64(HaxeInt64abs.xor(x.i64,y.i64));
}
public static inline function compare(x:GOint64,y:GOint64):Int {
	return HaxeInt64abs.compare(x.i64,y.i64);
}
public static inline function ucompare(x:GOint64,y:GOint64):Int {
	return HaxeInt64abs.ucompare(x.i64,y.i64);
}
}
#else
*/
