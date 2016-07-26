package MailClassifier;

import java.io.IOException;
import java.io.OutputStream;
import java.io.InputStream;
import java.io.OutputStreamWriter;
import java.net.InetSocketAddress;
import java.util.Collection;
import java.util.Iterator;
import java.util.List;
import java.util.ListIterator;
import java.util.ArrayList;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;

import org.apache.commons.io.IOUtils;
import org.deeplearning4j.berkeley.Pair;

import org.json.*;
public class Server {
    public static Classifier classifier = new Classifier();
    public static void main(String[] args) throws Exception {
        try
        {
            classifier.load();
        }
        catch (Exception e)
        {
            classifier.train();
            classifier.save();
        }

        int port = 8099;
        HttpServer server = HttpServer.create(new InetSocketAddress(port), 0);
        server.createContext("/classify", new ClassifyHandler());
        server.setExecutor(null); // creates a default executor
        server.start();

        System.out.println("Server started on port "+port);
    }

    static class ClassifyHandler implements HttpHandler {

        public void handle(HttpExchange t) throws IOException {
            InputStream is = t.getRequestBody();
            String request = IOUtils.toString(is, "UTF-8");
            IOUtils.closeQuietly(is);

            List<Pair<String,Double>> result = classifier.classify(request);

            JSONArray out = new JSONArray(result);
            String response = out.toString();

            t.sendResponseHeaders(200, response.getBytes().length);
            OutputStream os = t.getResponseBody();
            os.write(response.getBytes());
            os.close();
            System.out.println("Successfully received classification request.");
        }
    }

}
