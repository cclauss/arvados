class Trait < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  serialize :properties, Hash

  api_accessible :user, extend: :common do |t|
    t.add :name
    t.add :properties
  end
end
