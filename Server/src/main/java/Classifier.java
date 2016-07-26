
package MailClassifier;
import org.canova.api.util.ClassPathResource;
import org.deeplearning4j.berkeley.Pair;
import org.deeplearning4j.models.word2vec.Word2Vec;
import MailClassifier.tools.FileLabelAwareIterator;
import MailClassifier.tools.LabelSeeker;
import MailClassifier.tools.MeansBuilder;
import org.deeplearning4j.models.embeddings.inmemory.InMemoryLookupTable;
import org.deeplearning4j.models.paragraphvectors.ParagraphVectors;
import org.deeplearning4j.models.word2vec.VocabWord;
import org.deeplearning4j.text.documentiterator.LabelAwareIterator;
import org.deeplearning4j.text.documentiterator.LabelledDocument;
import org.deeplearning4j.text.tokenization.tokenizer.preprocessor.CommonPreprocessor;
import org.deeplearning4j.text.tokenization.tokenizerfactory.DefaultTokenizerFactory;
import org.deeplearning4j.text.tokenization.tokenizerfactory.TokenizerFactory;
import org.deeplearning4j.models.embeddings.loader.WordVectorSerializer;
import org.nd4j.linalg.api.ndarray.INDArray;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import MailClassifier.tools.Preprocessor;

import java.io.*;
import java.util.List;

public class Classifier {

    private static final Logger log = LoggerFactory.getLogger(Classifier.class);

    private Word2Vec paragraphVectors;
    private LabelAwareIterator iterator;
    private TokenizerFactory tokenizer;

    public void train() throws Exception
    {
        File dir = new File("../Client/trainingData");
        // build a iterator for our dataset
        iterator = new FileLabelAwareIterator.Builder()
                .addSourceFolder(dir)
                .build();

        tokenizer = new DefaultTokenizerFactory();
        tokenizer.setTokenPreProcessor(new CommonPreprocessor());

        // ParagraphVectors training configuration
        paragraphVectors = new ParagraphVectors.Builder()
                .learningRate(0.025)
                .minLearningRate(0.001)
                .batchSize(1000)
                .epochs(20)
                .iterate(iterator)
                .trainWordVectors(true)
                .tokenizerFactory(tokenizer)
                .build();

        // Start model training
        paragraphVectors.fit();

    }

    public void save() throws Exception
    {
        //WordVectorSerializer.writeFullModel(paragraphVectors, "model.bin");
    }

    public void load() throws Exception
    {
        paragraphVectors = WordVectorSerializer.loadFullModel("model.bin");

        File dir = new File("trainingData");
        // build a iterator for our dataset
        iterator = new FileLabelAwareIterator.Builder()
                .addSourceFolder(dir)
                .build();

        tokenizer = new DefaultTokenizerFactory();
        tokenizer.setTokenPreProcessor(new CommonPreprocessor());
    }

    public List<Pair<String, Double>> classify(String in)
    {
        MeansBuilder meansBuilder = new MeansBuilder((InMemoryLookupTable<VocabWord>) paragraphVectors.getLookupTable(), tokenizer);
        LabelSeeker seeker = new LabelSeeker(iterator.getLabelsSource().getLabels(), (InMemoryLookupTable<VocabWord>)  paragraphVectors.getLookupTable());

        LabelledDocument document = new LabelledDocument();
        document.setContent(Preprocessor.preprocess(in));

        INDArray documentAsCentroid = meansBuilder.documentAsVector(document);
        List<Pair<String, Double>> scores = seeker.getScores(documentAsCentroid);

        return scores;
    }
}
