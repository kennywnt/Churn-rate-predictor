package com.example.api;

import com.intuit.karate.junit5.Karate;

class PredictApiRunner {
    @Karate.Test
    Karate testPredict() {
        return Karate.run("predict").relativeTo(getClass());
    }
}
