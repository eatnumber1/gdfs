extern crate fuse;
extern crate getopts;
extern crate regex;

use fuse::Filesystem;
use getopts::Options;
use regex::Regex;
use std::env;
use std::ffi::OsStr;
use std::process::exit;
use std::os::unix::ffi::OsStrExt;

struct GdFS;

impl Filesystem for GdFS {
}

fn usage(program: &str, opts: Options) {
	let usage_text = format!("Usage: {} [options] <mountpount>", program);
	print!("{}", opts.usage(&usage_text));
}

fn parse_fuse_args(fuseopts: &Vec<String>) -> Vec<String> {
	let mut opts = Vec::new();
	let re = Regex::new(r"\s+").unwrap();

	for fuseopt in fuseopts {
		for optpart in fuseopt.splitn(1, &re) {
			opts.push(optpart.to_string());
		}
	}

	opts
}

fn prep_fuseargs(items: &Vec<String>) -> Vec<&OsStr> {
	let mut ret = Vec::new();

	for item in items {
		ret.push(OsStr::from_bytes(item.as_bytes()));
	}

	ret
}

fn main() {
	let args: Vec<String>  = env::args().collect();
	let program = args[0].clone();

	let mut opts = Options::new();
	opts.optflag("h", "help", "print this help menu");
	opts.optmulti("", "fuseopt",
		"one option/'option value' pair to pass to fuse implementation",
		"option");

	let matches = match opts.parse(&args[1..]) {
		Ok(m) => { m },
		Err(f) => { panic!(f.to_string()) }
	};

	if matches.opt_present("h") {
		usage(&program, opts);
		exit(1);
	}

	let fuseargs = if matches.opt_present("fuseopt") {
		parse_fuse_args(&matches.opt_strs("fuseopt"))
	} else {
		Vec::new()
	};

	let mountpoint = if ! matches.free.is_empty() {
		matches.free[0].clone()
	} else {
		usage(&program, opts);
		exit(1);
	};

	fuse::mount(GdFS, &mountpoint, &prep_fuseargs(&fuseargs)[..]);
}
