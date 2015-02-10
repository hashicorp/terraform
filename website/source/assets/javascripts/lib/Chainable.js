(function(){

var Chainable = function(engine){
	this.engine = engine;
	this._chain = [];
	this._updateTimer = this._updateTimer.bind(this);
	this._cycle = this._cycle.bind(this);
};

Chainable.prototype._running = false;

Chainable.prototype._updateTimer = function(tick){
	this._timer += tick;
	if (this._timer >= this._timerMax) {
		this.resetTimer();
		this._cycle();
	}
};

Chainable.prototype.resetTimer = function(){
	this.engine.updateChainTimer = undefined;
	this._timer = 0;
	this._timerMax = 0;
	return this;
};

Chainable.prototype.start = function(){
	if (this._running || !this._chain.length) {
		return this;
	}
	this._running = true;
	return this._cycle();
};

Chainable.prototype.reset = function(){
	if (!this._running) {
		return this;
	}
	this.resetTimer();
	this._timer = 0;
	this._running = false;
	return this;
};

Chainable.prototype._cycle = function(){
	var current;
	if (!this._chain.length) {
		return this.reset();
	}

	current = this._chain.shift();

	if (current.type === 'function') {
		current.func.apply(current.scope, current.args);
		current = null;
		return this._cycle();
	}
	if (current.type === 'wait') {
		this.resetTimer();
		// Convert timer to seconds
		this._timerMax = current.time / 1000;
		this.engine.updateChainTimer = this._updateTimer;
		current = null;
	}

	return this;
};

Chainable.prototype.then = Chainable.prototype.exec = function(func, scope, args){
	this._chain.push({
		type  : 'function',

		func  : func,
		scope : scope || window,
		args  : args  || []
	});

	return this.start();
};

Chainable.prototype.wait = function(time){
	this._chain.push({
		type : 'wait',
		time : time
	});

	return this.start();
};

window.Chainable = Chainable;

})();
