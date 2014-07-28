(function(
	Particle,
	Engine,
	Vector
){

Particle.Fixed = function(width, height){
	var targetX, targetY;

	this.radius = Engine.getRandomFloat(0.1, 1);
	// this.fillA = 'rgba(136,67,237,' + Engine.getRandomFloat(0.4, 0.5) + ')';
	// this.fillB = 'rgba(136,67,237,' + Engine.getRandomFloat(0.51, 0.6) + ')';
	this.fillA = '#3a1066';
	this.fillB = '#561799';
	this.frameMax = Engine.getRandomInt(4, 10);

	this.max = {
		x: width  + this.maxRadius,
		y: height + this.maxRadius
	};

	this.min = {
		x: 0 - this.maxRadius,
		y: 0 - this.maxRadius
	};

	targetX = Engine.getRandomInt(0 + this.radius, width  + this.radius);
	targetY = Engine.getRandomInt(0 + this.radius, height + this.radius);

	this.pos = new Vector(targetX, targetY);
};

Engine.Particle.Fixed.prototype = {

	radius: 1,

	pos: {
		x: 0,
		y: 0
	},

	frame: 0,
	showA: false,

	update: function(engine){
		this.frame++;
		if (this.frame > this.frameMax) {
			this.frame = 0;
			this.showA = !this.showA;
		}
		return this;
	},

	draw: function(ctx, scale){
		// Draw a circle - far less performant
		ctx.beginPath();
		ctx.arc(
			this.pos.x * scale >> 0,
			this.pos.y * scale >> 0,
			this.radius * scale,
			0,
			Math.PI * 2,
			false
		);
		if (this.showA) {
			ctx.fillStyle = this.fillA;
		} else {
			ctx.fillStyle = this.fillB;
		}
		ctx.fill();

		return this;
	}

};

})(window.Engine.Particle, window.Engine, window.Vector);
