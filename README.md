# AI Processing

Download pre-trained models:
 - yolov8n.pt
  - Export to onnx `yolo export model=path/to/best.pt format=onnx opset=12`
# Export

One of the option

`conda env export > environment.yml`
`conda list --explicit > spec_file_root.txt`

# Update
Choose one

`conda env update --name root --file environment.yml`
`conda create --name myenv2 --file spec_file.txt`


# Example output

https://github.com/user-attachments/assets/d10d34a4-d059-4db3-b170-7cd5675129bf

