(function(
	Engine,
	Vector
){

Engine.Particle = function(width, height){
	var side, targetX, targetY;
	this.accel = Vector.coerce(this.accel);
	this.vel   = Vector.coerce(this.vel);
	this.pos   = new Vector(
		width  / 2,
		height / 2
	);

	this.maxRadius = Engine.getRandomFloat(0.1, 2.5);
	this.maxSpeed = Engine.getRandomFloat(0.01, 1000);

	this.max = {
		x: width  + this.maxRadius,
		y: height + this.maxRadius
	};

	this.min = {
		x: 0 - this.maxRadius,
		y: 0 - this.maxRadius
	};

	// Pick a random target
	side = Engine.getRandomInt(0, 3);
	if (side === 0 || side === 2) {
		targetY = (side === 0) ? (0 - this.maxRadius) : (height + this.maxRadius);
		targetX = Engine.getRandomInt(0 - this.maxRadius, width + this.maxRadius);
	} else {
		targetY = Engine.getRandomInt(0 - this.maxRadius, height + this.maxRadius);
		targetX = (side === 3) ? (0 - this.maxRadius) : (width + this.maxRadius);
	}

	this.target = new Vector(targetX, targetY);
	this.maxDistance = this.distanceTo(this.target);

	// this.fillA = 'rgba(136,67,237,' + Engine.getRandomFloat(0.7, 0.8) + ')';
	// this.fillB = 'rgba(136,67,237,1)';
	// this.fillA = '#651bb3';
	// this.fillB = '#9027ff';
	this.fillA = '#8750c2';
	this.fillB = '#b976ff';
	// b976ff
	this.frameMax = Engine.getRandomInt(1, 5);
};

Engine.Particle.prototype = {

	radius: 1,

	frame: 0,
	showA: false,

	accel: {
		x: 0,
		y: 0
	},

	vel: {
		x: 0,
		y: 0
	},

	pos: {
		x: 0,
		y: 0
	},

	opacity: 1,

	maxSpeed: 1500,
	maxForce: 1500,

	update: function(engine){
		var distancePercent;

		this.accel.mult(0);
		this.seek();

		this.vel
			.add(this.accel)
			.limit(this.maxSpeed);

		this.pos.add(Vector.mult(this.vel, engine.tick));

		if (
			this.pos.x < this.min.x ||
			this.pos.x > this.max.x ||
			this.pos.y < this.min.y ||
			this.pos.y > this.max.y
		) {
			this.kill(engine);
		}

		distancePercent = (this.maxDistance - this.distanceTo(this.target)) / this.maxDistance;
		this.radius = Math.max(0.1, this.maxRadius * distancePercent);

		this.frame++;
		if (this.frame > this.frameMax) {
			this.frame = 0;
			this.showA = !this.showA;
		}

		return this;
	},

	seek: function(){
		var desired, steer;

		desired = Vector.sub(this.target, this.pos)
			.normalize()
			.mult(this.maxSpeed);

		steer = Vector
			.sub(desired, this.vel)
			.limit(this.maxForce);

		this.applyForce(steer);
	},

	draw: function(ctx, scale){
		if (this.radius < 0.25) {
			return;
		}

		if (this.showA) {
			ctx.fillStyle = this.fillA;
		} else {
			ctx.fillStyle = this.fillB;
		}

		// Draw a square - very performant
		ctx.fillRect(
			this.pos.x * scale >> 0,
			this.pos.y * scale >> 0,
			this.radius * scale,
			this.radius * scale
		);

		// Draw a circle - far less performant
		// ctx.beginPath();
		// ctx.arc(
		//     this.pos.x * scale,
		//     this.pos.y * scale,
		//     this.radius * scale,
		//     0,
		//     Math.PI * 2,
		//     false
		// );
		// ctx.fill();

		return this;
	},

	applyForce: function(force){
		this.accel.add(force);
		return this;
	},

	kill: function(engine){
		engine._deferredParticles.push(this);
		return this;
	},

	distanceTo: function(target) {
		var xd = this.pos.x - target.x;
		var yd = this.pos.y - target.y;
		return Math.sqrt(xd * xd + yd * yd );
	}
};

})(window.Engine, window.Vector);
