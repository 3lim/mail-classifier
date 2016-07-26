package MailClassifier.tools;

import org.jsoup.Jsoup;
import org.jsoup.examples.HtmlToPlainText;

public class Preprocessor {
    public static String preprocess(String in)
    {
        String plain = new HtmlToPlainText().getPlainText(Jsoup.parse(in));
        return plain;
    }
}
