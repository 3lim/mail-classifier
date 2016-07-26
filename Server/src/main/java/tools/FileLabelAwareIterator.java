package MailClassifier.tools;

import lombok.NonNull;
import org.apache.commons.io.IOUtils;
import org.deeplearning4j.text.documentiterator.LabelAwareIterator;
import org.deeplearning4j.text.documentiterator.LabelledDocument;
import org.deeplearning4j.text.documentiterator.LabelsSource;

import java.io.File;
import java.io.FileInputStream;
import java.util.ArrayList;
import java.util.List;
import java.util.Queue;
import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.atomic.AtomicInteger;

import org.json.*;

public class FileLabelAwareIterator implements LabelAwareIterator {
    protected List<File> files;
    protected Queue<String> currentDocs = new ArrayBlockingQueue<String>(200);
    protected String currentLabel = new String();
    protected AtomicInteger position = new AtomicInteger(0);
    protected LabelsSource labelsSource;

    /*
        Please keep this method protected, it's used in tests
     */
    protected FileLabelAwareIterator() {

    }

    protected FileLabelAwareIterator(@NonNull List<File> files, @NonNull LabelsSource source) {
        this.files = files;
        this.labelsSource = source;
    }

    public boolean hasNextDocument() {
        return !currentDocs.isEmpty() || position.get() < files.size();
    }


    public LabelledDocument nextDocument() {
        if(currentDocs.isEmpty())
        {
            File fileToRead = files.get(position.getAndIncrement());
            currentLabel = fileToRead.getName().substring(0, fileToRead.getName().lastIndexOf('.'));
            try {
                FileInputStream is = new FileInputStream(fileToRead);
                String docs = IOUtils.toString(is, "UTF-8");
                IOUtils.closeQuietly(is);

                JSONArray documents = new JSONArray(docs);
                for (int i=0; i<documents.length(); ++i)
                {
                    String document = documents.getJSONObject(i).getString("Answer");
                    currentDocs.add(Preprocessor.preprocess(document));
                }
            } catch (Exception e) {
                throw new RuntimeException(e);
            }
        }

        String content = currentDocs.poll();
        LabelledDocument document = new LabelledDocument();
        document.setContent(content);
        document.setLabel(currentLabel);
        return document;
    }

    public void reset() {
        position.set(0);
    }

    public LabelsSource getLabelsSource() {
        return labelsSource;
    }

    public static class Builder {
        protected List<File> foldersToScan = new ArrayList<File>();

        public Builder() {

        }

        public Builder addSourceFolder(@NonNull File folder) {
            foldersToScan.add(folder);
            return this;
        }

        public FileLabelAwareIterator build() {
            // search for all files in all folders provided
            List<File> fileList = new ArrayList<File>();
            List<String> labels = new ArrayList<String>();

            for (File file: foldersToScan) {
                if (!file.isDirectory()) continue;

                File[] files = file.listFiles();
                if (files == null || files.length ==0 ) continue;


                for (File fileLabel: files) {

                    String label = fileLabel.getName().substring(0, fileLabel.getName().lastIndexOf('.'));
                    if (!labels.contains(label)) labels.add(label);

                    fileList.add(fileLabel);
                }
            }
            LabelsSource source = new LabelsSource(labels);
            FileLabelAwareIterator iterator = new FileLabelAwareIterator(fileList, source);

            return iterator;
        }
    }
}