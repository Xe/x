use std::{io, fs::File, str, thread, time};
use std::io::prelude::*;

fn main() -> io::Result<()> {
    let stdin = io::stdin(); // We get `Stdin` here.
    let mut fout = File::create("localhost:1997")?;
    
    print!("input> ");
    io::stdout().lock().flush()?;
    let mut buf = String::new();
    stdin.read_line(&mut buf)?;
    write!(fout, "{}", buf)?;

    let ten_millis = time::Duration::from_millis(10);
    thread::sleep(ten_millis);

    let mut buf = Vec::new();
    fout.read_to_end(&mut buf)?;
    let buf = unsafe { str::from_utf8_unchecked(&buf) };
    print!("{}", buf);
    io::stdout().lock().flush()?;

    Ok(())
}
