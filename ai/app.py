import os
import string
import joblib
from flask import Flask, request, jsonify
from flask_cors import CORS

MODELS_DIR = os.path.join(os.path.dirname(__file__), 'models') # Path relative to app.py
VECTORIZER_PATH = os.path.join(MODELS_DIR, 'tfidf_vectorizer.joblib')
MODEL_PATH = os.path.join(MODELS_DIR, 'news_classifier_model.joblib')

app = Flask(__name__)
CORS(app, resources={r"/suggest-category": {"origins": "*"}})

vectorizer = None
model = None
try:
    if not os.path.exists(VECTORIZER_PATH):
        raise FileNotFoundError(f"Vectorizer not found at {VECTORIZER_PATH}")
    if not os.path.exists(MODEL_PATH):
        raise FileNotFoundError(f"Model not found at {MODEL_PATH}")
    vectorizer = joblib.load(VECTORIZER_PATH)
    model = joblib.load(MODEL_PATH)
    app.logger.info("Vectorizer and model loaded successfully.")
except Exception as e:
    app.logger.error(f"Error loading model or vectorizer: {e}", exc_info=True)

# Preprocessing Function (matches training)
PUNCTUATION_TRANSLATOR = str.maketrans('', '', string.punctuation)
def preprocess_text_flask(text_content: str) -> str:
    if not isinstance(text_content, str):
        app.logger.warn("preprocess_text_flask received non-string input, returning empty string.")
        return ""
    text_content = text_content.lower()
    text_content = text_content.translate(PUNCTUATION_TRANSLATOR)
    text_content = " ".join(text_content.split())
    return text_content

CATEGORY_MAP = {
    0: "World",   
    1: "Sports",  
    2: "Business",
    3: "Sci/Tech" 
}

@app.route('/suggest-category', methods=['POST'])
def suggest_category():
    if not model or not vectorizer:
        app.logger.error("Suggest category endpoint called but model/vectorizer not loaded.")
        return jsonify({"error": "Model or vectorizer not loaded properly on server"}), 500

    try:
        data = request.get_json()
        if not data:
            return jsonify({"error": "Request body must be JSON"}), 400
        if 'text' not in data:
            return jsonify({"error": "Missing 'text' field in JSON request"}), 400

        input_text = data['text']
        if not isinstance(input_text, str) or input_text.strip() == "":
            return jsonify({"error": "'text' field must be a non-empty string"}), 400

        app.logger.info(f"Received text for suggestion: {input_text[:100]}...")

        # Preprocess the input text
        processed_text = preprocess_text_flask(input_text)
        app.logger.info(f"Processed text: {processed_text[:100]}...")

        # Vectorize the processed text
        text_vector = vectorizer.transform([processed_text])
        app.logger.info("Text vectorized.")

        # Make a prediction
        prediction_index_array = model.predict(text_vector)
        if len(prediction_index_array) == 0:
            app.logger.error("Model prediction returned empty result.")
            return jsonify({"error": "Model prediction failed"}), 500
        
        prediction_index_raw = prediction_index_array[0]
        app.logger.info(f"Raw prediction index: {prediction_index_raw} (type: {type(prediction_index_raw)})")

        prediction_index = int(prediction_index_raw)

        # Map index to category name
        category_name = CATEGORY_MAP.get(prediction_index, "Unknown Category")
        app.logger.info(f"Predicted index: {prediction_index}, Category: {category_name}")

        return jsonify({
            "predicted_class_index": prediction_index,
            "predicted_category_name": category_name,
            "original_text_snippet": input_text[:100]
        })

    except Exception as e:
        app.logger.error(f"Error during prediction: {e}", exc_info=True)
        return jsonify({"error": "Error processing request on server", "details": str(e)}), 500

@app.route('/health', methods=['GET'])
def health_check():
    if model and vectorizer:
        return jsonify({"status": "AI Service OK", "model_loaded": True, "vectorizer_loaded": True}), 200
    else:
        return jsonify({"status": "AI Service ERROR", "model_loaded": False, "vectorizer_loaded": False}), 500


if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000, debug=os.environ.get("FLASK_DEBUG", "False").lower() == "true")