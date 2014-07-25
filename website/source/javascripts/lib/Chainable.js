(function(){

var Chainable = function(){
	this._chain = [];
	this._cycle = this._cycle.bind(this);
};

Chainable.prototype._running = false;

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
	clearTimeout(this._timer);
	this._timer = null;
	this._chain.length = 0;
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
		clearTimeout(this._timer);
		this._timer = setTimeout(this._cycle, current.time || 0);
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
