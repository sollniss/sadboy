package analyzer

func SetOpts(options func(o *Opts, caller *CallerOpts, callee *CalleeOpts)) func() {
	origOpts := opts
	origCallerOpts := callerOpts
	origCalleeOpts := calleeOpts

	opts = Opts{}
	callerOpts = CallerOpts{}
	calleeOpts = CalleeOpts{}
	options(&opts, &callerOpts, &calleeOpts)

	return func() {
		opts = origOpts
		callerOpts = origCallerOpts
		calleeOpts = origCalleeOpts
	}
}
