(function(
	Engine,
	Vector
){

Engine.Point.Puller = function(id, x, y){
	this.id = id;
	this.pos.x = x;
	this.pos.y = y;

	this.pos = Vector.coerce(this.pos);
	this.home = this.pos.clone();
	this.accel = Vector.coerce(this.accel);
	this.vel = Vector.coerce(this.vel);
};

Engine.Point.Puller.prototype = {

	fillStyle: null,
	defaultFillstyle: '#b976ff',
	chasingFillstyle: '#ff6b6b',

	radius: 1,

	maxSpeed: 160,
	maxForce: 50,

	pos: {
		x: 0,
		y: 0
	},

	accel: {
		x: 0,
		y: 0
	},

	vel: {
		x: 0,
		y: 0
	},

	aRad: 200,

	safety: 0.25,

	update: function(engine){
		var target = Vector.coerce(engine.mouse),
			distanceToMouse = this.distanceTo(target),
			toHome, mag, safety;
			// distanceToHome = this.distanceTo(this.home);

		this.accel.mult(0);

		if (distanceToMouse < this.aRad) {
			this._chasing = true;
			this.toChase(target);
			this.fillStyle = this.chasingFillstyle;
		} else {
			this._chasing = false;
			this.fillStyle = this.defaultFillstyle;
		}

		this.toChase(this.home, this.maxForce / 2);

		this.vel.add(this.accel);
		this.pos.add(
			Vector.mult(this.vel, engine.tick)
		);

		toHome = Vector.sub(this.home, this.pos);
		mag = toHome.mag();
		safety = this.aRad * (this.safety * 3);
		if (mag > this.aRad - safety) {
			toHome.normalize();
			toHome.mult(this.aRad - safety);
			this.pos = Vector.sub(this.home, toHome);
		}
	},

	toChase: function(target, maxForce){
		var desired, steer, distance, mult, safety;

		maxForce = maxForce || this.maxForce;

		target = Vector.coerce(target);
		desired = Vector.sub(target, this.pos);
		distance = desired.mag();
		desired.normalize();

		safety = this.aRad * this.safety;

		if (distance < safety) {
			mult = Engine.map(distance, 0, safety, 0, this.maxSpeed);
		} else if (distance > this.aRad - safety){
			mult = Engine.map(this.aRad - distance, 0, safety, 0, this.maxSpeed);
		} else {
			mult = this.maxSpeed;
		}

		desired.mult(mult);

		steer = Vector.sub(desired, this.vel);
		steer.limit(maxForce);
		this.accel.add(steer);
	},

	draw: function(ctx, scale){
		ctx.fillStyle = this.fillStyle;
		ctx.fillRect(
			(this.pos.x - this.radius / 2) * scale >> 0,
			(this.pos.y - this.radius / 2) * scale >> 0,
			this.radius * scale,
			this.radius * scale
		);
	},

	distanceTo: function(target) {
		var xd = this.home.x - target.x;
		var yd = this.home.y - target.y;
		return Math.sqrt(xd * xd + yd * yd );
	}
};

})(
	window.Engine,
	window.Vector
);
