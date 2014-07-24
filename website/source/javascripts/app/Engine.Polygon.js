(function(
	Engine,
	Vector
){

Engine.Polygon = function(a, b, c, color){
	this.a = a;
	this.b = b;
	this.c = c;

	this.color = color;
	this.maxS = this.color.s;
	this.maxL = this.color.l;

	// this.color.s = 0;
	this.color.l = 0;

	this.start = Date.now() / 1000;

	this.fillStyle = this.hslaTemplate.substitute(this.color);

	// this.up = !!Engine.getRandomInt(0,1);
	// this.hueShiftSpeed = 15;
	// this.toColor = {
	//     a: 1
	// };
};

Engine.Polygon.prototype = {

	rgbaTemplate: 'rgba({r},{g},{b},{a})',
	hslaTemplate: 'hsla({h},{s}%,{l}%,{a})',

	hueShiftSpeed: 20,
	duration: 3,
	delay: 2.5,

	// Determine color fill?
	update: function(engine){
		var delta;

		delta = engine.now - this.start;

		if (
			delta > this.delay &&
			delta < this.delay + this.duration + 1 &&
			this.color.l < this.maxL
		) {
			// this.color.s = this.maxS * delta / this.duration;
			this.color.l = this.maxL * (delta - this.delay) / this.duration;

			if (this.color.l > this.maxL) {
				// this.color.s = this.maxS;
				this.color.l = this.maxL;
			}

			this.fillStyle = this.hslaTemplate.substitute(this.color);
		}
	},

	draw: function(ctx, scale){
		ctx.beginPath();
		ctx.moveTo(
			this.a.pos.x * scale,
			this.a.pos.y * scale
		);
		ctx.lineTo(
			this.b.pos.x * scale,
			this.b.pos.y * scale
		);
		ctx.lineTo(
			this.c.pos.x * scale,
			this.c.pos.y * scale
		);
		ctx.closePath();
		ctx.fillStyle   = this.fillStyle;
		ctx.lineWidth = 0.25 * scale;
		ctx.strokeStyle = this.fillStyle;
		ctx.fill();
		ctx.stroke();
	}

};

})(window.Engine, window.Vector);
