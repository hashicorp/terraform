(function(
	Engine,
	Vector
){

Engine.Polygon = function(a, b, c, color, simple){
	this.a = a;
	this.b = b;
	this.c = c;

	this.color = Engine.clone(color);
	this.simple = simple;

	if (this.simple) {
		this.strokeColor = this.color;
	} else {
		this.maxL = this.color.l;
		this.strokeColor = {
			h: this.color.h,
			s: 0,
			l: 100,
			a: 1
		};
		this.color.l = 0;
	}

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
			this.color.l < this.maxL
		) {
			this.color.l = this.maxL * (delta - this.delay) / this.duration;

			this.strokeColor.s = this.color.s * (delta - this.delay) / this.duration;
			this.strokeColor.l = (this.maxL - 100) * (delta - this.delay) / this.duration + 100;

			if (this.color.l > this.maxL) {
				this.color.l = this.maxL;
			}

			this.strokeStyle = this.hslaTemplate.substitute(this.strokeColor);
			this.fillStyle = this.hslaTemplate.substitute(this.color);
		}
	},

	draw: function(ctx, scale){
		ctx.beginPath();
		ctx.moveTo(
			this.a.pos.x * scale >> 0,
			this.a.pos.y * scale >> 0
		);
		ctx.lineTo(
			this.b.pos.x * scale >> 0,
			this.b.pos.y * scale >> 0
		);
		ctx.lineTo(
			this.c.pos.x * scale >> 0,
			this.c.pos.y * scale >> 0
		);
		ctx.closePath();
		if (!this.noFill) {
			ctx.fillStyle   = this.fillStyle;
			ctx.fill();
		}
		if (!this.simple) {
			ctx.lineWidth = 0.25 * scale;
			ctx.strokeStyle = this.strokeStyle;
			ctx.stroke();
		}
	}

};

})(window.Engine, window.Vector);
