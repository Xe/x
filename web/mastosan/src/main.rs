use lol_html::{element, html_content::ContentType, HtmlRewriter, Settings};
use std::env::args;
use std::io::{self, prelude::*, stdin, stdout};

fn main() -> io::Result<()> {
    let mode = match args().nth(1) {
        None => {
            return Err(io::Error::new(
                io::ErrorKind::InvalidInput,
                "usage: mastosan [markdown|slackdown|text]",
            ))
        }
        Some(mode) => mode,
    };

    let mut output = Vec::new();
    let mut rewriter = HtmlRewriter::new(
        Settings {
            element_content_handlers: vec![
                element!("span", |el| {
                    el.remove_and_keep_content();
                    Ok(())
                }),
                element!("p", |el| {
                    el.append("\n\n", ContentType::Html);
                    el.remove_and_keep_content();
                    Ok(())
                }),
                element!("br", |el| {
                    el.append("\n\n", ContentType::Html);
                    el.remove_and_keep_content();
                    Ok(())
                }),
                element!("a[href]", |el| {
                    let href = el.get_attribute("href").unwrap();
                    match mode.as_str() {
                        "slackdown" => {
                            el.prepend("|", ContentType::Html);
                            el.prepend(&href, ContentType::Html);
                            el.prepend("<", ContentType::Html);
                            el.append(">", ContentType::Html);
                        }
                        "markdown" => {
                            el.prepend("](", ContentType::Html);
                            el.prepend(&href, ContentType::Html);
                            el.prepend("[", ContentType::Html);
                            el.append(")", ContentType::Html);
                        }
                        _ => {}
                    }
                    el.remove_and_keep_content();

                    Ok(())
                }),
            ],
            ..Settings::default()
        },
        |c: &[u8]| output.extend_from_slice(c),
    );

    let mut input = Vec::new();
    let mut fin = stdin().lock();
    fin.read_to_end(&mut input)?;

    rewriter.write(&input).unwrap();
    rewriter.end().unwrap();

    let mut fout = stdout().lock();
    fout.write(&output)?;

    Ok(())
}
