(function(global){ 'use strict';

var Vector = function(x, y){
	this.x = x || 0;
	this.y = y || 0;
};

Vector.prototype = {

	clone: function(){
		return new Vector(this.x, this.y);
	},

	add: function(vec){
		this.x += vec.x;
		this.y += vec.y;
		return this;
	},

	sub: function(vec){
		this.x -= vec.x;
		this.y -= vec.y;
		return this;
	},

	subVal: function(val){
		this.x -= val;
		this.y -= val;
		return this;
	},

	mult: function(mul){
		this.x *= mul;
		this.y *= mul;
		return this;
	},

	div: function(div){
		if (div === 0) {
			return this;
		}
		this.x /= div;
		this.y /= div;
		return this;
	},

	mag: function(){
		return Math.sqrt(
			this.x * this.x +
			this.y * this.y
		);
	},

	limit: function(max){
		if (this.mag() > max) {
			this.normalize();
			this.mult(max);
		}
		return this;
	},

	normalize: function(){
		var mag = this.mag();
		if (mag === 0) {
			return this;
		}
		this.div(mag);
		return this;
	},

	heading: function(){
		return Math.atan2(this.y, this.x);
	},

	set: function(vec){
		this.x = vec.x;
		this.y = vec.y;
		return this;
	}

};

Vector.add = function(vec1, vec2){
	return vec1.clone().add(vec2.clone());
};

Vector.sub = function(vec1, vec2){
	return vec1.clone().sub(vec2.clone());
};

Vector.mult = function(vec, mult){
	return vec.clone().mult(mult);
};

Vector.div = function(vec, div){
	return vec.clone().div(div);
};

// Ripped from processing
Vector.random2D = function(){
	var angle = Math.random(0, 1) * Math.PI * 2;
	return new Vector(Math.cos(angle), Math.sin(angle));
};

Vector.coerce = function(obj){
	return new Vector(obj.x, obj.y);
};

global.Vector = Vector;

})(this);
