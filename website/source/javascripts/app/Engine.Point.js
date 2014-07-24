(function(
	Engine,
	Vector
){

Engine.Point = function(id, x, y, width, height){
	this.id = id;
	this.pos = new Vector(x, y);
	this.target = this.pos.clone();
	this.pos.x = width  / 2;
	this.pos.y = height / 2;
	this.accel = Vector.coerce(this.accel);
	this.vel = Vector.coerce(this.vel);

	this.pos.add({
		x: (Engine.getRandomFloat(0, 6) - 3),
		y: (Engine.getRandomFloat(0, 6) - 3)
	});

	// Physics randomness
	// this.stiffness = Engine.getRandomFloat(2, 5);
	// this.stiffness = Engine.getRandomFloat(0.4, 0.8);
	this.stiffness = Engine.getRandomFloat(3, 6);
	this.friction  = Engine.getRandomFloat(0.15, 0.3);
};

Engine.Point.prototype = {

	radius: 1,

	stiffness : 0.5,
	// friction  : 0.00001,
	friction  : 0.01,
	threshold : 0.03,

	pos: {
		x: 0,
		y: 0
	},

	accel: {
		x: 0,
		y: 0
	},

	vel : {
		x: 0,
		y: 0
	},

	target: {
		x: 0,
		y: 0
	},

	update: function(engine){
		var newAccel;

		newAccel = Vector.sub(this.target, this.pos)
			.mult(this.stiffness)
			.sub(Vector.mult(this.vel, this.friction));

		this.accel.set(newAccel);

		this.vel.add(this.accel);

		this.pos.add(
			Vector.mult(this.vel, engine.tick)
		);
	},

	draw: function(ctx, scale){
		ctx.beginPath();
		ctx.arc(
			this.pos.x * scale,
			this.pos.y * scale,
			this.radius * scale,
			0,
			Math.PI * 2,
			false
		);
		ctx.fillStyle = '#ffffff';
		ctx.fill();
	}

};

})(window.Engine, window.Vector);
