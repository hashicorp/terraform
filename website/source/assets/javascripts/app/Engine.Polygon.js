(function(
	Engine,
	Vector
){

Engine.Polygon = function(a, b, c, color, strokeColor){
	this.a = a;
	this.b = b;
	this.c = c;

	this.color = Engine.clone(color);
	this.strokeColor = strokeColor ? Engine.clone(strokeColor) : Engine.clone(color);

	if (strokeColor) {
		this.strokeColor = Engine.clone(strokeColor);
	} else {
		this.strokeColor = Engine.clone(color);
	}

	this.strokeWidth = 0.25;
	this.maxStrokeS = this.strokeColor.s;
	this.maxStrokeL = this.strokeColor.l;
	this.maxColorL  = this.color.l;

	this.strokeColor.s = 0;
	this.strokeColor.l = 100;
	this.color.l = 0;

	this.fillStyle = this.hslaTemplate.substitute(this.color);
	this.strokeStyle = this.hslaTemplate.substitute(this.strokeColor);
};

Engine.Polygon.prototype = {

	rgbaTemplate: 'rgba({r},{g},{b},{a})',
	hslaTemplate: 'hsla({h},{s}%,{l}%,{a})',

	hueShiftSpeed: 20,
	duration: 2,
	delay: 0,
	start: 0,

	// Determine color fill?
	update: function(engine){
		var delta;

		if (this.simple) {
			return;
		}

		this.start += engine.tick;

		delta = this.start;

		if (
			delta > this.delay &&
			delta < this.delay + this.duration + 1 &&
			this.color.l < this.maxColorL
		) {
			this.color.l = this.maxColorL * (delta - this.delay) / this.duration;

			this.strokeColor.s = this.maxStrokeS * (delta - this.delay) / this.duration;
			this.strokeColor.l = (this.maxStrokeL - 100) * (delta - this.delay) / this.duration + 100;

			this.strokeWidth = 1.5 * (delta - this.delay) / this.duration + 0.25;

			if (this.color.l > this.maxColorL) {
				this.color.l = this.maxColorL;
				this.strokeColor.l = this.maxStrokeL;
				this.strokeWidth = 1.5;
			}

			this.strokeStyle = this.hslaTemplate.substitute(this.strokeColor);
			this.fillStyle = this.hslaTemplate.substitute(this.color);
		}
	}

};

})(window.Engine, window.Vector);
